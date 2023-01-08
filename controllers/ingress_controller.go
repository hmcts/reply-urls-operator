/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"github.com/hmcts/reply-urls-operator/api/v1alpha1"
	azureGraph "github.com/hmcts/reply-urls-operator/controllers/pkg/azure"
	"github.com/hmcts/reply-urls-operator/controllers/pkg/secretsHandler"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	ingressClassNameField   = "spec.ingressClassName"
	ingressClassFilterField = "spec.ingressClassFilter"
)

var (
	defaultDomainFilter = ".*"
	workerLog           = ctrl.Log
	ingressList         = v1.IngressList{}
)

// IngressReconciler reconciles an Ingress object
type IngressReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/status,verbs=get
//+kubebuilder:rbac:groups=appregistrations.azure.hmcts.net,resources=replyurlsyncs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=appregistrations.azure.hmcts.net,resources=replyurlsyncs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=appregistrations.azure.hmcts.net,resources=replyurlsyncs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Ingress object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile

func (r *IngressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var (
		hosts []string

		ingress      = v1.Ingress{}
		replyURLSync = v1alpha1.ReplyURLSync{}
	)

	_ = log.FromContext(ctx)

	err := r.Get(ctx, req.NamespacedName, &ingress)

	if err != nil {
		if errors.IsNotFound(err) {
			result, err := r.cleanReplyURLSyncList()
			if err != nil {
				return result, err
			}

			return ctrl.Result{}, nil
		} else {
			return ctrl.Result{}, err
		}
	}

	ingressClassName := func() *string {
		var ingressAnnotation string
		if ingress.Spec.IngressClassName != nil {
			return ingress.Spec.IngressClassName
		} else if ingress.Annotations["kubernetes.io/ingress.class"] != "" {
			ingressAnnotation = ingress.Annotations["kubernetes.io/ingress.class"]
			return &ingressAnnotation
		}

		return nil
	}()

	for _, rules := range ingress.Spec.Rules {
		hosts = append(hosts, rules.Host)
	}

	// If ingressClassName or hosts empty, ignore event
	if ingressClassName == nil || hosts == nil {
		return ctrl.Result{}, nil
	}

	if replyURLSyncListAll, err := r.listReplyURLSync(nil); len(replyURLSyncListAll.Items) == 0 {
		workerLog.Info("Missing resource",
			"ReplyURLSync", "No ReplyURLSync resources found",
		)
	} else if err != nil {
		return ctrl.Result{}, err
	}

	replyURLSyncList, err := r.listReplyURLSync(ingressClassName)

	if err != nil {
		return ctrl.Result{}, err
	} else if len(replyURLSyncList.Items) == 0 {
		return ctrl.Result{}, nil
	}

	// Find replyURLSync with matching ingressClassName
	for _, replyURLSyncItem := range replyURLSyncList.Items {
		if *replyURLSyncItem.Spec.IngressClassFilter == *ingressClassName {
			replyURLSync = replyURLSyncItem
			break
		} else {
			return ctrl.Result{}, nil
		}
	}

	syncSpec := replyURLSync.Spec
	clientSecret := syncSpec.ClientSecret

	clientSecretCreds := azureGraph.ClientSecretCredentials{}

	if syncSpec.ClientID != nil {
		clientSecretCreds.ClientID = *syncSpec.ClientID
	} else {
		workerLog.Info("Missing configuration", "ClientID was not found in sync config")
		return ctrl.Result{}, nil
	}

	if clientSecret.EnvVarClientSecret != nil {
		if clientSecretValue, found := os.LookupEnv(*clientSecret.EnvVarClientSecret); found {

			clientSecretCreds.ClientSecret = clientSecretValue
		} else {
			workerLog.Info(fmt.Sprintf("%s environment variable not found", clientSecret.EnvVarClientSecret))
			return ctrl.Result{}, nil
		}
	} else if clientSecret.KeyVaultClientSecret.SecretName != "" && clientSecret.KeyVaultClientSecret.KeyVaultName != "" {
		secretName := clientSecret.KeyVaultClientSecret.SecretName
		keyVaultName := clientSecret.KeyVaultClientSecret.KeyVaultName

		// Get Secret from key vault
		secretsList, err := secretsHandler.GetSecretsFromVault(
			[]string{secretName},
			keyVaultName,
		)
		if err != nil {
			return ctrl.Result{}, err
		}
		if len(secretsList.Secrets) > 0 {

			for _, secret := range secretsList.Secrets {
				if secret.Name == clientSecret.KeyVaultClientSecret.SecretName {
					clientSecretCreds.ClientSecret = secret.Value
				}
			}
		} else {
			workerLog.Info("secret" + secretName + "not found")

		}
	}

	if replyURLSync.Spec.TenantID != nil {
		clientSecretCreds.TenantID = *replyURLSync.Spec.TenantID
	} else {
		workerLog.Info("Missing configuration", "TenantID was not found in sync config")
		return ctrl.Result{}, nil
	}

	if replyURLSync.Spec.ObjectID == nil {
		fnf := azureGraph.FieldNotFoundError{}
		fnf.SetResource(replyURLSync.Kind + "./" + replyURLSync.Name)
		fnf.SetField(".spec.objectID")
		return ctrl.Result{}, fnf
	}

	if replyURLSync.Spec.DomainFilter == nil {
		replyURLSync.Spec.DomainFilter = &defaultDomainFilter
	}

	result, err := azureGraph.ProcessHost(
		&v1.IngressList{
			Items: []v1.Ingress{
				ingress,
			},
		},
		replyURLSync.Spec,
		clientSecretCreds,
	)

	if err != nil {
		return ctrl.Result{}, err
	}

	return result, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IngressReconciler) SetupWithManager(mgr ctrl.Manager) error {

	ingress := &v1.Ingress{}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), ingress, ingressClassNameField, func(rawObj client.Object) []string {
		ingressClassName := rawObj.(*v1.Ingress).Spec.IngressClassName

		if ingressClassName == nil {
			return []string{}
		}
		return []string{*ingressClassName}

	}); err != nil {
		return err
	}

	replyURLSync := &v1alpha1.ReplyURLSync{}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), replyURLSync, ingressClassFilterField, func(rawObj client.Object) []string {
		ingressClass := rawObj.(*v1alpha1.ReplyURLSync).Spec.IngressClassFilter

		if ingressClass == nil {
			return []string{}
		}
		return []string{*ingressClass}

	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Ingress{}).
		Complete(r)
}

func (r *IngressReconciler) cleanReplyURLSyncList() (result ctrl.Result, err error) {

	replyURLSyncList, err := r.listReplyURLSync(nil)
	if err != nil {
		return ctrl.Result{}, err
	}
	/*
		Compare current list of redirect urls and remove any of
		them that don't have a corresponding ingress host on
		the cluster.
	*/
	for _, syncer := range replyURLSyncList.Items {
		var (
			opts              []client.ListOption
			clientSecretCreds azureGraph.ClientSecretCredentials
		)

		syncSpec := syncer.Spec
		clientSecret := syncSpec.ClientSecret

		if clientSecret.EnvVarClientSecret != nil {
			if clientSecretValue, found := os.LookupEnv(*clientSecret.EnvVarClientSecret); found {
				clientSecretCreds.ClientSecret = clientSecretValue
			}
		} else if clientSecret.KeyVaultClientSecret.SecretName != "" && clientSecret.KeyVaultClientSecret.KeyVaultName != "" {
			secretName := syncSpec.ClientSecret.KeyVaultClientSecret.SecretName
			keyVaultName := syncSpec.ClientSecret.KeyVaultClientSecret.KeyVaultName

			// Get Secret from key vault
			secretsList, err := secretsHandler.GetSecretsFromVault(
				[]string{secretName},
				keyVaultName,
			)

			for _, secret := range secretsList.Secrets {
				if secret.Name == syncSpec.ClientSecret.KeyVaultClientSecret.SecretName {
					clientSecretCreds.ClientSecret = secret.Value
				}
			}

			if err != nil {
				workerLog.Info("unable to get client secret: " + err.Error())
				return ctrl.Result{}, nil
			}

		}

		if syncSpec.ClientID != nil {
			clientSecretCreds.ClientID = *syncSpec.ClientID
		} else {
			workerLog.Info("Missing clientID from replyURLSyncSpec")
			return ctrl.Result{}, nil
		}

		if syncSpec.TenantID != nil {
			clientSecretCreds.TenantID = *syncSpec.TenantID
		} else {
			workerLog.Info("Missing tenantID from replyURLSyncSpec")
			return ctrl.Result{}, nil
		}

		if syncSpec.ObjectID == nil {
			fnf := azureGraph.FieldNotFoundError{}
			fnf.SetResource(syncer.Kind + "./" + syncer.Name)
			fnf.SetField(".spec.objectID")
			return ctrl.Result{}, fnf
		}

		err = r.List(context.TODO(), &ingressList, opts...)
		if err != nil {
			workerLog.Error(err, "Couldn't list ingress")
		}

		ingresses, err := azureGraph.FilterAndFormatIngressHosts(
			&ingressList,
			*syncSpec.DomainFilter,
			*syncSpec.IngressClassFilter,
		)

		if err != nil {
			return ctrl.Result{}, err
		}

		appRegPatchOptions := azureGraph.PatchOptions{
			IngressHosts: ingresses,
			Syncer:       syncer,
		}

		removedURLS, err := azureGraph.PatchAppRegistration(clientSecretCreds, appRegPatchOptions)
		if err != nil {
			return ctrl.Result{}, err
		}

		if removedURLS != nil {
			workerLog.Info("Reply URLs removed",
				"URLs", removedURLS,
				"object id", *syncSpec.ObjectID,
				"ingressClassName", *syncSpec.IngressClassFilter,
			)
		}
	}

	return ctrl.Result{}, nil
}

func (r *IngressReconciler) listReplyURLSync(ingressClassName *string) (replyURLSyncList *v1alpha1.ReplyURLSyncList, err error) {
	var opts []client.ListOption
	if ingressClassName != nil {
		opts = []client.ListOption{
			client.MatchingFields{ingressClassFilterField: *ingressClassName},
		}
	}
	replyURLSyncList = &v1alpha1.ReplyURLSyncList{}

	err = r.List(context.TODO(), replyURLSyncList, opts...)
	if err != nil {
		return nil, err
	}
	return replyURLSyncList, nil
}

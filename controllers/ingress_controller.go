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
	"github.com/modern-go/reflect2"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	domainFilter            = ".*"
	ingressClassNameField   = "spec.ingressClassName"
	ingressClassFilterField = "spec.ingressClassFilter"
)

// IngressReconciler reconciles an Ingress object
type IngressReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=networking.k8s.io.hmcts.net,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io.hmcts.net,resources=ingresses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=networking.k8s.io.hmcts.net,resources=ingresses/finalizers,verbs=update

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
		//workerLog       = ctrl.Log
		hosts           []string
		ingress         = v1.Ingress{}
		redirectUriSync = v1alpha1.RedirectUriSync{}
	)

	_ = log.FromContext(ctx)

	err := r.Get(ctx, req.NamespacedName, &ingress)

	if err != nil {
		if errors.IsNotFound(err) {
			result, err := r.cleanRedirectUriSyncList()
			if err != nil {
				return result, err
			}

			return ctrl.Result{}, nil
		} else {
			return ctrl.Result{}, err
		}
	}

	ingressClassName := ingress.Spec.IngressClassName

	for _, rules := range ingress.Spec.Rules {
		hosts = append(hosts, rules.Host)
	}

	// If ingressClassName or hosts empty, ignore event
	if ingressClassName == nil || hosts == nil {
		return ctrl.Result{}, nil
	}

	redirectUriSyncList, err := r.listRedirectUriSync(ingressClassName)
	if err != nil {
		return ctrl.Result{}, err
	}
	// Find redirectUriSync with matching ingressClassName
	for _, redirectUriSyncItem := range redirectUriSyncList.Items {
		if *redirectUriSyncItem.Spec.IngressClassFilter == *ingressClassName {
			redirectUriSync = redirectUriSyncItem

		} else {
			return ctrl.Result{}, nil
		}
	}

	fnf := azureGraph.FieldNotFoundError{}
	fnf.SetResource(redirectUriSync.Kind + "./" + redirectUriSync.Name)

	if !reflect2.IsNil(*redirectUriSync.Spec.DomainFilter) {
		*redirectUriSync.Spec.DomainFilter = domainFilter
	}

	result, err := azureGraph.ProcessHost(hosts, redirectUriSync.Spec)
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

	redirectUriSync := &v1alpha1.RedirectUriSync{}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), redirectUriSync, ingressClassFilterField, func(rawObj client.Object) []string {
		ingressClass := rawObj.(*v1alpha1.RedirectUriSync).Spec.IngressClassFilter

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

func (r *IngressReconciler) cleanRedirectUriSyncList() (result ctrl.Result, err error) {

	var (
		workerLog   = ctrl.Log
		ingressList = v1.IngressList{}
	)

	redirectUriSyncList, err := r.listRedirectUriSync(nil)
	if err != nil {
		return ctrl.Result{}, err
	}
	/*
		Compare current list of redirect uris and remove any of
		them that don't have a corresponding ingress host on
		the cluster.
	*/
	for _, syncer := range redirectUriSyncList.Items {
		var opts []client.ListOption
		syncSpec := syncer.Spec

		if syncSpec.IngressClassFilter != nil {
			// get list of ingresses on cluster
			opts = []client.ListOption{
				client.MatchingFields{ingressClassNameField: *syncSpec.IngressClassFilter},
			}
		}

		err = r.List(context.TODO(), &ingressList, opts...)
		if err != nil {
			fmt.Println(err)
		}

		ingresses, err := azureGraph.FilterIngresses(&ingressList, syncSpec.DomainFilter)
		if err != nil {
			return ctrl.Result{}, err
		}

		appRegPatchOptions := azureGraph.PatchOptions{
			IngressHosts: ingresses,
			Syncer:       syncer,
		}
		removedURLS, err := azureGraph.PatchAppRegistration(appRegPatchOptions)
		if err != nil {
			return ctrl.Result{}, err
		}

		workerLog.Info("Host removed",
			"hosts", removedURLS,
			"object id", *syncSpec.ClientID,
			"ingressClassName", *syncSpec.IngressClassFilter,
		)

	}

	return ctrl.Result{}, nil
}

func (r *IngressReconciler) listRedirectUriSync(ingressClassName *string) (redirectUriSyncList *v1alpha1.RedirectUriSyncList, err error) {
	var opts []client.ListOption
	if ingressClassName != nil {
		opts = []client.ListOption{
			client.MatchingFields{ingressClassFilterField: *ingressClassName},
		}
	}
	redirectUriSyncList = &v1alpha1.RedirectUriSyncList{}

	err = r.List(context.TODO(), redirectUriSyncList, opts...)
	if err != nil {
		return nil, err
	}
	return redirectUriSyncList, nil
}

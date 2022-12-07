package azureGraph

import (
	"context"
	"github.com/go-openapi/swag"
	"github.com/hmcts/reply-urls-operator/api/v1alpha1"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	graph "github.com/microsoftgraph/msgraph-sdk-go/models"
	v1 "k8s.io/api/networking/v1"
	"regexp"
	ctrl "sigs.k8s.io/controller-runtime"
)

func getApplication(appId string, graphClient *msgraphsdk.GraphServiceClient) (appObject graph.Applicationable, err error) {
	application, err := graphClient.ApplicationsById(appId).Get(context.TODO(), nil)
	if err != nil {
		return application, err
	}

	return application, err
}

func GetReplyURLs(appId string, graphClient *msgraphsdk.GraphServiceClient) (replyURLs []string, err error) {
	appObject, err := getApplication(appId, graphClient)
	if err != nil {
		return nil, err
	}
	replyURLs = appObject.GetWeb().GetRedirectUris()
	return replyURLs, nil
}

func PatchAppReplyURLs(appId string, urls []string, graphClient *msgraphsdk.GraphServiceClient) error {
	// Patch Application
	requestBody := graph.NewApplication()
	app := graph.NewWebApplication()

	app.SetRedirectUris(urls)
	requestBody.SetWeb(app)

	_, err := graphClient.ApplicationsById(appId).Patch(context.TODO(), requestBody, nil)

	if err != nil {
		return err
	}

	return nil
}

func PatchAppRegistration(patchOptions PatchOptions) (removedURLS []string, err error) {
	var (
		newRedirectURLS []string

		syncer                 = patchOptions.Syncer
		syncSpec               = syncer.Spec
		syncerFullResourceName = syncer.Name
		replyURLFilter         = syncSpec.ReplyURLFilter
	)

	azureAppClient, err := CreateClient()
	if err != nil {
		return nil, err
	}

	if syncSpec.ObjectID == nil {
		fnfErr := FieldNotFoundError{
			Field:    ".spec.objectID",
			Resource: syncerFullResourceName,
		}
		return nil, fnfErr
	}

	urls, err := GetReplyURLs(*syncSpec.ObjectID, azureAppClient)
	if err != nil {
		return nil, err
	}

	for _, url := range urls {
		if swag.ContainsStrings(patchOptions.IngressHosts, url) {
			newRedirectURLS = append(newRedirectURLS, url)
		} else {
			/*
				If a replyURL filter isn't set, delete all reply urls that do not
				exist as an ingress host on the cluster

				If it is set and the url matches with the filter delete it
				If it is set and the url doesn't match do not delete it
			*/
			if replyURLFilter == nil {
				removedURLS = append(removedURLS, url)
			} else {
				if matched, err := regexp.MatchString(*replyURLFilter, url); err != nil {
					return nil, err
				} else if matched {
					removedURLS = append(removedURLS, url)
				} else {
					newRedirectURLS = append(newRedirectURLS, url)
				}
			}
		}
	}

	if len(removedURLS) == 0 {
		return nil, nil
	}

	if len(newRedirectURLS) == 0 {
		newRedirectURLS = []string{}
	}

	if err := PatchAppReplyURLs(*syncSpec.ObjectID, newRedirectURLS, azureAppClient); err != nil {
		return nil, err
	}
	return removedURLS, nil
}

func ProcessHost(ingresses *v1.IngressList, syncSpec v1alpha1.ReplyURLSyncSpec) (result ctrl.Result, err error) {

	var (
		urls           []string
		azureAppClient *msgraphsdk.GraphServiceClient
		workerLog      = ctrl.Log
	)

	if azureAppClient, err = CreateClient(); err != nil {
		return ctrl.Result{}, err
	}

	formattedURLs, err := FilterAndFormatIngressHosts(ingresses, *syncSpec.DomainFilter, *syncSpec.IngressClassFilter)

	if err != nil {
		workerLog.Error(err, "Unable to filter lists")
	}

	for _, url := range formattedURLs {

		if urls, err = GetReplyURLs(*syncSpec.ObjectID, azureAppClient); err != nil {
			return ctrl.Result{}, err
		} else {
			if !swag.ContainsStrings(urls, url) {
				urls = append(urls, url)
				if err := PatchAppReplyURLs(*syncSpec.ObjectID, urls, azureAppClient); err != nil {
					return ctrl.Result{}, err
				}
				workerLog.Info("Reply URL added",
					"URL", url,
					"object id", *syncSpec.ObjectID, "ingressClassName", *syncSpec.IngressClassFilter)
			}
		}
	}

	return ctrl.Result{}, nil
}

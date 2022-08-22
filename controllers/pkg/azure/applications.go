package azureGraph

import (
	"fmt"
	"github.com/go-openapi/swag"
	"github.com/hmcts/reply-urls-operator/api/v1alpha1"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	graph "github.com/microsoftgraph/msgraph-sdk-go/models"
	"regexp"
	ctrl "sigs.k8s.io/controller-runtime"
)

func getApplication(appId string, graphClient *msgraphsdk.GraphServiceClient) (appObject graph.Applicationable, err error) {
	application, err := graphClient.ApplicationsById(appId).Get()
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

	err := graphClient.ApplicationsById(appId).Patch(requestBody)
	if err != nil {
		return err
	}
	return nil
}

func PatchAppRegistration(patchOptions PatchOptions) (removedURLS []string, err error) {
	syncSpec := patchOptions.Syncer.Spec
	syncerResource := patchOptions.Syncer.Name
	var newRedirectURLS []string
	azureAppClient, err := CreateClient()
	if err != nil {
		return nil, err
	}

	if syncSpec.ClientID == nil {
		fnfErr := FieldNotFoundError{
			Field:    ".spec.clientID",
			Resource: syncerResource,
		}
		return nil, fnfErr
	}

	urls, err := GetReplyURLs(*syncSpec.ClientID, azureAppClient)
	if err != nil {
		return nil, err
	}

	for _, url := range urls {
		if swag.ContainsStrings(patchOptions.IngressHosts, url) {
			newRedirectURLS = append(newRedirectURLS, url)
		} else {
			removedURLS = append(removedURLS, url)
		}
	}

	if err := PatchAppReplyURLs(*syncSpec.ClientID, newRedirectURLS, azureAppClient); err != nil {
		return nil, err
	}
	return removedURLS, nil
}

func ProcessHost(hosts []string, syncSpec v1alpha1.ReplyURLSyncSpec) (result ctrl.Result, err error) {

	var (
		urls           []string
		azureAppClient *msgraphsdk.GraphServiceClient
		workerLog      = ctrl.Log
	)

	if azureAppClient, err = CreateClient(); err != nil {
		return ctrl.Result{}, err
	}

	for _, host := range hosts {

		if isMatch, err := regexp.MatchString(*syncSpec.DomainFilter, host); err != nil {
			return ctrl.Result{}, err

		} else if !isMatch {
			// Host doesn't match filter so it can be ignored
			return ctrl.Result{}, nil

		}

		if urls, err = GetReplyURLs(*syncSpec.ClientID, azureAppClient); err != nil {
			return ctrl.Result{}, err
		} else {
			hostFormatted := fmt.Sprintf("https://%s", host)
			if !swag.ContainsStrings(urls, hostFormatted) {
				urls = append(urls, hostFormatted)
				if err := PatchAppReplyURLs(*syncSpec.ClientID, urls, azureAppClient); err != nil {
					return ctrl.Result{}, err
				}
				workerLog.Info("Host added",
					"host", hostFormatted,
					"object id", *syncSpec.ClientID, "ingressClassName", *syncSpec.IngressClassFilter)
			}
		}
	}

	return ctrl.Result{}, nil
}

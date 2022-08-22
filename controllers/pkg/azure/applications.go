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

func GetRedirectURIs(appId string, graphClient *msgraphsdk.GraphServiceClient) (redirectURIs []string, err error) {
	appObject, err := getApplication(appId, graphClient)
	if err != nil {
		return nil, err
	}
	redirectURIs = appObject.GetWeb().GetRedirectUris()
	return redirectURIs, nil
}

func PatchAppRedirectURIs(appId string, uris []string, graphClient *msgraphsdk.GraphServiceClient) error {
	// Patch Application
	requestBody := graph.NewApplication()
	app := graph.NewWebApplication()

	app.SetRedirectUris(uris)
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

	uris, err := GetRedirectURIs(*syncSpec.ClientID, azureAppClient)
	if err != nil {
		return nil, err
	}

	for _, uri := range uris {
		if swag.ContainsStrings(patchOptions.IngressHosts, uri) {
			newRedirectURLS = append(newRedirectURLS, uri)
		} else {
			removedURLS = append(removedURLS, uri)
		}
	}

	if err := PatchAppRedirectURIs(*syncSpec.ClientID, newRedirectURLS, azureAppClient); err != nil {
		return nil, err
	}
	return removedURLS, nil
}

const (
	//domainFilter            = ".*"
	ingressClassNameField = "spec.ingressClassName"
	//ingressClassFilterField = "spec.ingressClassFilter"
)

func ProcessHost(hosts []string, syncSpec v1alpha1.RedirectUriSyncSpec) (result ctrl.Result, err error) {

	var (
		uris           []string
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

		if uris, err = GetRedirectURIs(*syncSpec.ClientID, azureAppClient); err != nil {
			return ctrl.Result{}, err
		} else {
			hostFormatted := fmt.Sprintf("https://%s", host)
			if !swag.ContainsStrings(uris, hostFormatted) {
				uris = append(uris, hostFormatted)
				if err := PatchAppRedirectURIs(*syncSpec.ClientID, uris, azureAppClient); err != nil {
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

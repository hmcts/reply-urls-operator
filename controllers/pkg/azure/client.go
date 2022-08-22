package azureGraph

import (
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	a "github.com/microsoft/kiota-authentication-azure-go"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
)

func CreateClient() (client *msgraphsdk.GraphServiceClient, error error) {
	auth, err := GraphAuth()
	if err != nil {
		fmt.Printf("Error with authentication: %v\n", err)
		return nil, err
	}

	adapter, err := msgraphsdk.NewGraphRequestAdapter(auth)
	if err != nil {
		fmt.Printf("Error creating adapter: %v\n", err)
		return nil, err
	}
	graphClient := msgraphsdk.NewGraphServiceClient(adapter)
	return graphClient, nil
}

func GraphAuth() (graphCreds *a.AzureIdentityAuthenticationProvider, err error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		fmt.Printf("Error with credentials: %v", err)
		return nil, err
	}

	auth, err := a.NewAzureIdentityAuthenticationProviderWithScopes(cred, []string{"https://graph.microsoft.com/.default"})
	if err != nil {
		fmt.Printf("Error authentication provider: %v\n", err)
		return auth, err
	}
	return auth, nil
}

package azureGraph

import (
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	a "github.com/microsoft/kiota-authentication-azure-go"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
)

func GraphAuth(c *ClientSecretCredentials) (graphCreds *a.AzureIdentityAuthenticationProvider, err error) {
	cred, err := azidentity.NewClientSecretCredential(c.TenantID, c.ClientID, c.ClientSecret, nil)
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

func CreateClient(c *ClientSecretCredentials) (client *msgraphsdk.GraphServiceClient, error error) {
	auth, err := GraphAuth(c)
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

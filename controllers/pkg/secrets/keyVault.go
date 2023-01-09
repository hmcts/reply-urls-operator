package secrets

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
)

var (
	tokenOptions = policy.TokenRequestOptions{
		Scopes: []string{"https://graph.microsoft.com/.default"},
	}
)

func keyVaultAuthManagedIdentity(keyVaultURI string) (client *azsecrets.Client, err error) {

	credential, err := azidentity.NewManagedIdentityCredential(nil)
	if err != nil {
		return nil, err
	}

	// Test getting a token
	_, err = credential.GetToken(context.TODO(), tokenOptions)

	if err != nil {
		return nil, err
	}

	client, err = azsecrets.NewClient(keyVaultURI, credential, nil)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func keyVaultAuthAzureCLI(keyVaultURI string) (client *azsecrets.Client, err error) {
	credential, err := azidentity.NewAzureCLICredential(nil)
	if err != nil {
		return nil, err
	}

	// Test getting a token
	_, err = credential.GetToken(context.TODO(), tokenOptions)

	if err != nil {
		return nil, err
	}

	client, err = azsecrets.NewClient(keyVaultURI, credential, nil)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func GetSecretFromVault(secretName string, keyVaultName string) (secret *string, err error) {

	var (
		keyVaultURI = fmt.Sprintf("https://%s.vault.azure.net/", keyVaultName)
		client      = &azsecrets.Client{}
	)

	client, err = keyVaultAuthManagedIdentity(keyVaultURI)
	if err != nil {

		client, err = keyVaultAuthAzureCLI(keyVaultURI)
		if err != nil {
			return nil, err
		}
	}

	// empty string version gets the latest version of the secret
	version := ""
	resp, err := client.GetSecret(context.TODO(), secretName, version, nil)
	if err != nil {
		return nil, err
	}

	secret = resp.Value

	return secret, nil
}

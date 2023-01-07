package secretsHandler

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
)

func keyVaultAuthManagedIdentity() (credential *azidentity.ManagedIdentityCredential, AuthErr error) {

	credential, err := azidentity.NewManagedIdentityCredential(nil)
	if err != nil {
		return nil, err
	}

	return credential, nil
}

func keyVaultAuthAzureCLI() (credential *azidentity.AzureCLICredential, AuthErr error) {
	credential, err := azidentity.NewAzureCLICredential(nil)
	if err != nil {
		return nil, err
	}

	return credential, nil
}

func GetSecretsFromVault(secretNameList []string, keyVaultName string) (*SecretList, error) {

	secretList := &SecretList{}
	var credential azcore.TokenCredential
	keyVaultURI := fmt.Sprintf("https://%s.vault.azure.net/", keyVaultName)

	credential, err := keyVaultAuthAzureCLI()
	if err != nil {
		credential, err = keyVaultAuthManagedIdentity()
		if err != nil {

			return nil, err
		}
	}
	client, err := azsecrets.NewClient(keyVaultURI, credential, nil)
	if err != nil {
		return nil, err
	}

	for _, secretName := range secretNameList {
		// Get a secret. An empty string version gets the latest version of the secret.
		version := ""
		resp, err := client.GetSecret(context.TODO(), secretName, version, nil)
		if err != nil {
			return nil, err
		}

		if *resp.Value != "" {
			secretList.Secrets = append(secretList.Secrets, Secret{Name: secretName, Value: *resp.Value})
		}
	}

	return secretList, nil
}

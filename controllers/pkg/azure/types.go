package azureGraph

import "github.com/hmcts/reply-urls-operator/api/v1alpha1"

type PatchOptions struct {
	IngressHosts []string
	Syncer       v1alpha1.RedirectUriSync
}

package azureGraph

import (
	"fmt"
	v1 "k8s.io/api/networking/v1"
	"regexp"
)

func FilterIngresses(ingressList *v1.IngressList, domainFilter *string) (ingressHosts []string, err error) {
	for _, ingressItem := range ingressList.Items {
		for _, rule := range ingressItem.Spec.Rules {
			if isMatch, err := regexp.MatchString(*domainFilter, rule.Host); err != nil {
				return nil, err
			} else if !isMatch {
				continue
			}

			// If ingress host matches domain regex add it to the list of ingresses that should be managed
			ingressHosts = append(ingressHosts, fmt.Sprintf("https://%s", rule.Host))

		}
	}
	return ingressHosts, nil
}

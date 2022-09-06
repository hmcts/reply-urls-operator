package azureGraph

import (
	"fmt"
	v1 "k8s.io/api/networking/v1"
	"regexp"
)

func FilterAndFormatIngresses(ingressList *v1.IngressList, domainFilter *string) (ingressHosts []string, err error) {
	for _, ingressItem := range ingressList.Items {
		for _, rule := range ingressItem.Spec.Rules {
			if isMatch, err := regexp.MatchString(*domainFilter, rule.Host); err != nil {
				return nil, err
			} else if !isMatch {
				continue
			}

			// If ingress host matches domain regex add it to the list of ingresses that should be managed
			ingressHosts = append(ingressHosts, fmt.Sprintf("https://%s/oauth-proxy/callback", rule.Host))

		}
	}
	return ingressHosts, nil
}

func FilterAndFormatStringList(stringList []string, filter string) (filteredStringList []string, err error) {
	for _, s := range stringList {
		if matchesFilter, err := regexp.MatchString(filter, s); err != nil {
			return nil, err

		} else if matchesFilter {
			filteredStringList = append(filteredStringList, s)
		}
	}
	return filteredStringList, nil
}

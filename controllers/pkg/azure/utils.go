package azureGraph

import (
	"fmt"
	v1 "k8s.io/api/networking/v1"
	"regexp"
)

func FilterAndFormatIngressHosts(ingressList *v1.IngressList, domainFilter string, ingressClassFilter string) (ingressHosts []string, err error) {
	for _, ingress := range ingressList.Items {

		/*
		  If either ingressClass name annotation and ingressClassName spec field don't
		  match the ingressClassNameFilter continue and filter
		*/

		if (ingress.Spec.IngressClassName == nil || *ingress.Spec.IngressClassName != ingressClassFilter) &&
			ingress.Annotations["kubernetes.io/ingress.class"] != ingressClassFilter {

			continue

		}
		for _, rule := range ingress.Spec.Rules {
			if isMatch, err := regexp.MatchString(domainFilter, rule.Host); err != nil {
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

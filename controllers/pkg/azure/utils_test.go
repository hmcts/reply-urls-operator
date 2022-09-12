package azureGraph

import (
	v1 "k8s.io/api/networking/v1"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"strings"
	"testing"
)

func TestFilterAndFormatIngressHosts(t *testing.T) {
	var (
		domainFilter           = ".*.sandbox.platform.hmcts.net"
		ingressClassNameFilter = "traefik"
	)

	noIngressClassAnnotation := v1meta.ObjectMeta{
		Annotations: map[string]string{},
	}

	withIngressClassAnnotation := v1meta.ObjectMeta{
		Annotations: map[string]string{
			"kubernetes.io/ingress.class": "traefik",
		},
	}

	wrongIngressClassAnnotation := v1meta.ObjectMeta{
		Annotations: map[string]string{
			"kubernetes.io/ingress.class": "wrong-traefik",
		},
	}

	ingressList := v1.IngressList{
		Items: []v1.Ingress{
			{
				ObjectMeta: noIngressClassAnnotation,
				Spec: v1.IngressSpec{
					Rules: []v1.IngressRule{
						{
							Host: "test-app-1.sandbox.platform.hmcts.net",
						},
					},
				},
			},
			{
				ObjectMeta: noIngressClassAnnotation,
				Spec: v1.IngressSpec{
					IngressClassName: &ingressClassNameFilter,
					Rules: []v1.IngressRule{
						{
							Host: "test-app-2.platform.hmcts.net",
						},
					},
				},
			},
			{
				ObjectMeta: withIngressClassAnnotation,
				Spec: v1.IngressSpec{
					Rules: []v1.IngressRule{
						{
							Host: "test-app-3.sandbox.platform.hmcts.net",
						},
					},
				},
			},
			{
				ObjectMeta: noIngressClassAnnotation,
				Spec: v1.IngressSpec{
					IngressClassName: &ingressClassNameFilter,
					Rules: []v1.IngressRule{
						{
							Host: "test-app-4.sandbox.platform.hmcts.net",
						},
					},
				},
			},
			{
				ObjectMeta: withIngressClassAnnotation,
				Spec: v1.IngressSpec{

					Rules: []v1.IngressRule{
						{
							Host: "test-app-5.staging.platform.hmcts.net",
						},
					},
				},
			},
			{
				ObjectMeta: withIngressClassAnnotation,
				Spec: v1.IngressSpec{
					IngressClassName: nil,
					Rules: []v1.IngressRule{
						{
							Host: "test-app-6.sandbox.platform.hmcts.net",
						},
					},
				},
			},
			{
				ObjectMeta: wrongIngressClassAnnotation,
				Spec: v1.IngressSpec{
					IngressClassName: nil,
					Rules: []v1.IngressRule{
						{
							Host: "test-app-7.sandbox.platform.hmcts.net",
						},
					},
				},
			},
		},
	}

	expectedList := []string{
		"https://test-app-3.sandbox.platform.hmcts.net/oauth-proxy/callback",
		"https://test-app-4.sandbox.platform.hmcts.net/oauth-proxy/callback",
		"https://test-app-6.sandbox.platform.hmcts.net/oauth-proxy/callback",
	}

	if list, _ := FilterAndFormatIngressHosts(
		&ingressList,
		domainFilter,
		ingressClassNameFilter,
	); !reflect.DeepEqual(list, expectedList) {
		t.Errorf("Result %v not equal to the expected result %v\nTest: %s\n",
			list, expectedList, strings.ToLower(t.Name()))
	}
}

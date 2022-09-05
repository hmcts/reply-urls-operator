package azureGraph

import (
	v1 "k8s.io/api/networking/v1"
	"reflect"
	"strings"
	"testing"
)

func TestFilterIngresses(t *testing.T) {
	ingressList := v1.IngressList{
		Items: []v1.Ingress{
			{
				Spec: v1.IngressSpec{Rules: []v1.IngressRule{
					{Host: "test-app-1.platform.hmcts.net"},
				}},
			},
			{
				Spec: v1.IngressSpec{Rules: []v1.IngressRule{
					{Host: "test-app-2.sandbox.platform.hmcts.net"},
				}},
			},
			{
				Spec: v1.IngressSpec{Rules: []v1.IngressRule{
					{Host: "test-app-3.platform.hmcts.net"},
				}},
			},
			{
				Spec: v1.IngressSpec{Rules: []v1.IngressRule{
					{Host: "test-app-4.sandbox.platform.hmcts.net"},
				}},
			},
			{
				Spec: v1.IngressSpec{Rules: []v1.IngressRule{
					{Host: "test-app-5.staging.platform.hmcts.net"},
				}},
			},
		},
	}
	domainFilter := ".*.sandbox.platform.hmcts.net"
	expectedList := []string{
		"https://test-app-2.sandbox.platform.hmcts.net/oauth-proxy/callback",
		"https://test-app-4.sandbox.platform.hmcts.net/oauth-proxy/callback",
	}

	if list, _ := FilterAndFormatIngresses(&ingressList, &domainFilter); !reflect.DeepEqual(list, expectedList) {
		t.Errorf("Result %v not equal to the expected result %v\nTest: %s\n",
			list, expectedList, strings.ToLower(t.Name()))
	}
}

func TestFilterStringList(t *testing.T) {
	filter := "sandbox.platform.hmcts.net"
	stringList := []string{
		"test-app-1.staging.platform.hmcts.net",
		"test-app-2.sandbox.platform.hmcts.net",
		"test-app-3.test.platform.hmcts.net",
		"test-app-4.sandbox.platform.hmcts.net",
		"test-app-5.sbox.platform.hmcts.net",
		"test-app-6.sandbox.platform.hmcts.net",
	}

	expectedStringList := []string{
		"test-app-2.sandbox.platform.hmcts.net",
		"test-app-4.sandbox.platform.hmcts.net",
		"test-app-6.sandbox.platform.hmcts.net",
	}

	if returnedStringList, _ := FilterAndFormatStringList(stringList, filter); !reflect.DeepEqual(returnedStringList, expectedStringList) {
		t.Errorf("Result %v not equal to the expected result %v\nTest: %s\n",
			returnedStringList, expectedStringList, strings.ToLower(t.Name()))
	}
}

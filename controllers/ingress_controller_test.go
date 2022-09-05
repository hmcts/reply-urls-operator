package controllers

import (
	"context"
	appregistrationsazurev1alpha1 "github.com/hmcts/reply-urls-operator/api/v1alpha1"
	azureGraph "github.com/hmcts/reply-urls-operator/controllers/pkg/azure"
	v1 "k8s.io/api/networking/v1"
	"os"
	"sort"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	clientID     = "2816f198-4c26-48bb-8732-e4ca72926ba7"
	objectID     = "850e80c0-e09e-489d-b12d-5e80cd1bca6a"
	tenantID     = "21ae17a1-694c-4005-8e0f-6a0e51c35a5f"
	domainFilter = ".*sandbox.platform.hmcts.net"
	ingressClass = "traefik"

	testIngresses = []ingresses{
		{
			name:             "test-app-1",
			host:             "test-app-1.sandbox.platform.hmcts.net",
			ingressClassName: ingressClass,
		},
		{
			name:             "test-app-2",
			host:             "test-app-2.sandbox.platform.hmcts.net",
			ingressClassName: ingressClass,
		},
		{
			name:             "test-app-3",
			host:             "test-app-3.sandbox.platform.hmcts.net",
			ingressClassName: "private",
		},
		{
			name:             "test-app-4",
			host:             "test-app-4.staging.platform.hmcts.net",
			ingressClassName: "private",
		},
		{
			name:             "test-app-5",
			host:             "test-app-5.sandbox.platform.hmcts.net",
			ingressClassName: ingressClass,
		},
		{
			name:             "test-app-6",
			host:             "test-app-6.staging.platform.hmcts.net",
			ingressClassName: "private",
		},
	}
)

const (
	ingressNamespace = "default"
)

type ingresses struct {
	name             string
	host             string
	ingressClassName string
}

var _ = Describe("ReplyURLSync Config", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		replyURLSyncName      = "test-reply-url-sync"
		replyURLSyncNamespace = "default"

		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	clientSecret := os.Getenv("TESTING_AZURE_CLIENT_SECRET")
	//tenantID := os.Getenv("TESTING_AZURE_TENANT_ID")
	//clientID := os.Getenv("TESTING_AZURE_CLIENT_ID")
	//objectID := os.Getenv("TESTING_AZURE_OBJECT_ID")

	os.Setenv("AZURE_CLIENT_SECRET", clientSecret)
	os.Setenv("AZURE_TENANT_ID", tenantID)
	os.Setenv("AZURE_CLIENT_ID", clientID)

	Context("When creating a replyURLSync", func() {
		It("It should get created", func() {
			By("By creating a new replyURLSync")
			ctx := context.Background()
			replyURLSync := &appregistrationsazurev1alpha1.ReplyURLSync{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "appregistrations.azure.hmcts.net/v1alpha1",
					Kind:       "ReplyURLSync",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      replyURLSyncName,
					Namespace: replyURLSyncNamespace,
				},
				Spec: appregistrationsazurev1alpha1.ReplyURLSyncSpec{
					TenantID:           &tenantID,
					ObjectID:           &objectID,
					ClientID:           &clientID,
					DomainFilter:       &domainFilter,
					IngressClassFilter: &ingressClass,
				},
			}
			Expect(k8sClient.Create(ctx, replyURLSync)).Should(Succeed())

			replyURLSyncLookupKey := types.NamespacedName{Name: replyURLSyncName, Namespace: replyURLSyncNamespace}
			createdReplyURLSync := &appregistrationsazurev1alpha1.ReplyURLSync{}

			// We'll need to retry getting this newly created CronJob, given that creation may not immediately happen.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, replyURLSyncLookupKey, createdReplyURLSync)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			// Let's make sure our Schedule string value was properly converted/handled.
			Expect(createdReplyURLSync.Spec.ClientID).Should(Equal(&clientID))
			Expect(createdReplyURLSync.Spec.ObjectID).Should(Equal(&objectID))
			Expect(createdReplyURLSync.Spec.TenantID).Should(Equal(&tenantID))
			Expect(createdReplyURLSync.Spec.IngressClassFilter).Should(Equal(&ingressClass))
			Expect(createdReplyURLSync.Spec.DomainFilter).Should(Equal(&domainFilter))

		})
	})

	Context("When creating ingresses", func() {
		It("should update the list of reply urls on the app registration", func() {
			By("the operator")
			ctx := context.Background()

			for _, testIngress := range testIngresses {
				ingress := &v1.Ingress{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "networking.k8s.io/v1",
						Kind:       "Ingress",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      testIngress.name,
						Namespace: ingressNamespace,
					},
					Spec: v1.IngressSpec{
						IngressClassName: &testIngress.ingressClassName,
						Rules: []v1.IngressRule{
							{Host: testIngress.host},
						},
					},
				}
				Expect(k8sClient.Create(ctx, ingress)).Should(Succeed())

				ingressLookupKey := types.NamespacedName{Name: ingress.Name, Namespace: ingressNamespace}
				createdIngress := &v1.Ingress{}

				Eventually(func() bool {
					err := k8sClient.Get(ctx, ingressLookupKey, createdIngress)
					if err != nil {
						return false
					}
					return true
				}, timeout, interval).Should(BeTrue())
				// Let's make sure our Schedule string value was properly converted/handled.
				Expect(createdIngress.Spec.IngressClassName).Should(Equal(&testIngress.ingressClassName))
				Expect(createdIngress.Spec.Rules[0].Host).Should(Equal(testIngress.host))
			}

		})
	})

	expectedHosts := []string{
		"https://test-app-1.sandbox.platform.hmcts.net/oauth-proxy/callback",
		"https://test-app-2.sandbox.platform.hmcts.net/oauth-proxy/callback",
		"https://test-app-5.sandbox.platform.hmcts.net/oauth-proxy/callback",
	}
	sort.Strings(expectedHosts)

	Context("When running the operator", func() {

		It("should update the list of reply urls on the app registration", func() {
			By("checking the ingresses on the cluster")

			Eventually(func() []string {
				client, _ := azureGraph.CreateClient()
				// Need to look at
				//if err != nil {
				//	return err
				//}
				replyURLHosts, _ := azureGraph.GetReplyURLs(objectID, client)
				sort.Strings(replyURLHosts)
				//if err != nil {
				//	return err
				//}

				return replyURLHosts
			}, timeout, interval).Should(Equal(expectedHosts))

		})

	})
})

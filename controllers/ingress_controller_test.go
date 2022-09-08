package controllers

import (
	"context"
	"fmt"
	"github.com/hmcts/reply-urls-operator/api/v1alpha1"
	azureGraph "github.com/hmcts/reply-urls-operator/controllers/pkg/azure"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/utils/strings/slices"
	"math/rand"
	"os"
	"sort"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	clientID              = "2816f198-4c26-48bb-8732-e4ca72926ba7"
	objectID              = "850e80c0-e09e-489d-b12d-5e80cd1bca6a"
	tenantID              = "21ae17a1-694c-4005-8e0f-6a0e51c35a5f"
	domainFilter          = ".*sandbox.platform.hmcts.net"
	ingressClass          = "traefik"
	replyURLSyncName      = "test-reply-url-sync"
	replyURLSyncNamespace = "default"

	replyURLSync = &v1alpha1.ReplyURLSync{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "appregistrations.azure.hmcts.net/v1alpha1",
			Kind:       "ReplyURLSync",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      replyURLSyncName,
			Namespace: replyURLSyncNamespace,
		},
		Spec: v1alpha1.ReplyURLSyncSpec{
			TenantID:           &tenantID,
			ObjectID:           &objectID,
			ClientID:           &clientID,
			DomainFilter:       &domainFilter,
			IngressClassFilter: &ingressClass,
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
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	clientSecret := os.Getenv("TESTING_AZURE_CLIENT_SECRET")

	err := os.Setenv("AZURE_CLIENT_SECRET", clientSecret)
	if err != nil {
		workerLog.Error(err, "Test Error")
	}

	err = os.Setenv("AZURE_TENANT_ID", tenantID)
	if err != nil {
		workerLog.Error(err, "Test Error")
	}
	err = os.Setenv("AZURE_CLIENT_ID", clientID)
	if err != nil {
		workerLog.Error(err, "Test Error")
	}

	// Test run ID is used to identify which urls to manage in tests

	testRunID, found := os.LookupEnv("GITHUB_EVENT_NUMBER")
	if found == false {
		testRunID = fmt.Sprintf("%d", rand.Intn(100))
	}

	var (
		expectedURLS = []string{
			"https://test-app-1-" + testRunID + ".sandbox.platform.hmcts.net/oauth-proxy/callback",
			"https://test-app-2-" + testRunID + ".sandbox.platform.hmcts.net/oauth-proxy/callback",
			"https://test-app-5-" + testRunID + ".sandbox.platform.hmcts.net/oauth-proxy/callback",
		}

		testIngresses = []ingresses{
			{
				name:             "test-app-1",
				host:             fmt.Sprintf("test-app-1-%s.sandbox.platform.hmcts.net", testRunID),
				ingressClassName: ingressClass,
			},
			{
				name:             "test-app-2",
				host:             fmt.Sprintf("test-app-2-%s.sandbox.platform.hmcts.net", testRunID),
				ingressClassName: ingressClass,
			},
			{
				name:             "test-app-3",
				host:             fmt.Sprintf("test-app-3-%s.ithc.platform.hmcts.net", testRunID),
				ingressClassName: "private",
			},
			{
				name:             "test-app-4",
				host:             fmt.Sprintf("test-app-4-%s.staging.platform.hmcts.net", testRunID),
				ingressClassName: "private",
			},
			{
				name:             "test-app-5",
				host:             fmt.Sprintf("test-app-5-%s.sandbox.platform.hmcts.net", testRunID),
				ingressClassName: ingressClass,
			},
			{
				name:             "test-app-6",
				host:             fmt.Sprintf("test-app-6-%s.platform.hmcts.net", testRunID),
				ingressClassName: "private",
			},
		}
	)
	Context("When creating a replyURLSync", func() {
		It("It should get created", func() {
			By("By creating a new replyURLSync")
			ctx := context.Background()

			Expect(k8sClient.Create(ctx, replyURLSync)).Should(Succeed())

			replyURLSyncLookupKey := types.NamespacedName{Name: replyURLSyncName, Namespace: replyURLSyncNamespace}
			createdReplyURLSync := &v1alpha1.ReplyURLSync{}

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

	Context("When creating a new reply url sync resource", func() {
		It("The app registrations list of urls for the pr should be empty", func() {
			By("By cleaning up the list")
			appRegPatchOptions := azureGraph.PatchOptions{
				IngressHosts: []string{""},
				Syncer:       *replyURLSync,
			}

			client, err := azureGraph.CreateClient()
			if err != nil {
				workerLog.Error(err, "Test Error")
			}

			replyURLS, err := azureGraph.GetReplyURLs(*appRegPatchOptions.Syncer.Spec.ObjectID, client)
			if err != nil {
				workerLog.Error(err, "Test Error")
			}

			var cleanedReplyURLS []string
			for _, url := range replyURLS {
				if !slices.Contains(expectedURLS, url) {
					cleanedReplyURLS = append(cleanedReplyURLS, url)
				}
			}

			err = azureGraph.PatchAppReplyURLs(*appRegPatchOptions.Syncer.Spec.ObjectID, cleanedReplyURLS, client)
			if err != nil {
				workerLog.Error(err, "Test Error")
			}

			Eventually(func() []string {
				var foundURLS = make([]string, 0)
				replyURLS, err := azureGraph.GetReplyURLs(*appRegPatchOptions.Syncer.Spec.ObjectID, client)
				if err != nil {
					workerLog.Error(err, "Test Error")
				}

				for _, url := range expectedURLS {
					if slices.Contains(replyURLS, url) {
						foundURLS = append(foundURLS, url)
					}
				}
				return foundURLS
			}, timeout, interval).Should(Equal([]string{}))

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

	Context("When running the operator", func() {

		It("should update the list of reply urls on the app registration", func() {
			By("checking the ingresses on the cluster")

			sort.Strings(expectedURLS)

			Eventually(func() (foundURLS []string) {
				client, err := azureGraph.CreateClient()
				if err != nil {
					workerLog.Error(err, "Test Error")
				}

				replyURLHosts, _ := azureGraph.GetReplyURLs(objectID, client)

				if err != nil {
					workerLog.Error(err, "Test Error")
				}

				//sort.Strings(replyURLHosts)
				for _, url := range expectedURLS {
					if slices.Contains(replyURLHosts, url) {
						foundURLS = append(foundURLS, url)
					}
				}

				return foundURLS
			}, timeout, interval).Should(Equal(expectedURLS))

		})

	})
})

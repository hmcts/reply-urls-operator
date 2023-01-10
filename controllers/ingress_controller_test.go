package controllers

import (
	"context"
	"fmt"
	"github.com/hmcts/reply-urls-operator/api/v1alpha1"
	azureGraph "github.com/hmcts/reply-urls-operator/controllers/pkg/azure"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/utils/strings/slices"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	clientID           = "2816f198-4c26-48bb-8732-e4ca72926ba7"
	objectID           = "850e80c0-e09e-489d-b12d-5e80cd1bca6a"
	envVarClientSecret = "TESTING_AZURE_CLIENT_SECRET"
	tenantID           = "21ae17a1-694c-4005-8e0f-6a0e51c35a5f"
	ingressClass       = "traefik"

	replyURLSyncName      = "test-reply-url-sync"
	replyURLSyncNamespace = "default"
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
	if clientSecret == "" {

		workerLog.Info("Environment variable missing for credentials. Attempting to use another Auth Method",
			"var", "TESTING_AZURE_CLIENT_SECRET",
		)
	}

	// Test run ID is used to identify which urls to manage in tests

	testRunID, found := os.LookupEnv("GITHUB_EVENT_NUMBER")
	if found == false {
		testRunID = "local"
	}

	var (
		domainFilter   = ".*" + testRunID + ".sandbox.platform.hmcts.net"
		replyURLFilter = ".*" + testRunID + ".sandbox.platform.hmcts.net"

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
				ReplyURLFilter:     &replyURLFilter,
				IngressClassFilter: &ingressClass,
				ClientSecret: &v1alpha1.ClientSecret{
					EnvVarClientSecret: &envVarClientSecret,
				},
			},
		}

		clientSecretCreds = azureGraph.ClientSecretCredentials{
			TenantID:     tenantID,
			ClientID:     clientID,
			ClientSecret: clientSecret,
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

		expectedURLS = []string{
			"https://test-app-1-" + testRunID + ".sandbox.platform.hmcts.net/oauth-proxy/callback",
			"https://test-app-2-" + testRunID + ".sandbox.platform.hmcts.net/oauth-proxy/callback",
			"https://test-app-5-" + testRunID + ".sandbox.platform.hmcts.net/oauth-proxy/callback",
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
				IngressHosts: []string{},
				Syncer:       *replyURLSync,
			}

			client, err := azureGraph.CreateClient(&clientSecretCreds)
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
			}, timeout, interval).Should(BeEmpty())

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

			Eventually(func() (foundURLS []string) {
				client, err := azureGraph.CreateClient(&clientSecretCreds)
				if err != nil {
					workerLog.Error(err, "Test Error")
				}

				replyURLHosts, _ := azureGraph.GetReplyURLs(objectID, client)

				if err != nil {
					workerLog.Error(err, "Test Error")
				}

				for _, url := range expectedURLS {

					if slices.Contains(replyURLHosts, url) {
						foundURLS = append(foundURLS, url)
					}
				}

				return foundURLS
			}, timeout, interval).Should(Equal(expectedURLS))

		})

	})

	Context("When deleting ingresses", func() {
		It("the ingresses hosts should be cleared from the app reg reply urls", func() {
			By("Deleting them")

			for _, testIngress := range testIngresses {
				ctx := context.Background()
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
							{
								Host: testIngress.host,
							},
						},
					},
				}
				Expect(k8sClient.Delete(ctx, ingress)).Should(Succeed())
			}

			Eventually(func() []v1.Ingress {
				ingressList := &v1.IngressList{}
				err := k8sClient.List(context.TODO(), ingressList, nil...)
				if err != nil {
					workerLog.Error(err, "Failed to get ingresses")
				}

				return ingressList.Items
			}, timeout, interval).Should(BeEmpty())

			Eventually(func() (foundURLS []string) {

				client, err := azureGraph.CreateClient(&clientSecretCreds)
				if err != nil {
					workerLog.Error(err, "Test Error")
				}

				replyURLS, err := azureGraph.GetReplyURLs(objectID, client)
				if err != nil {
					workerLog.Error(err, "Test Error")
				}

				for _, i := range expectedURLS {
					if slices.Contains(replyURLS, i) {
						foundURLS = append(foundURLS, i)

					}
				}

				return foundURLS
			}, timeout, interval).Should(BeEmpty())

		})
	})

})

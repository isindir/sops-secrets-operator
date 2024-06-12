package controllers_test

import (
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	isindirv1alpha3 "github.com/isindir/sops-secrets-operator/api/v1alpha3"
	controller "github.com/isindir/sops-secrets-operator/internal/controllers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"context"
	"path/filepath"
	"time"
)

var _ = Describe("SopssecretController", func() {
	TestSecretObject00 := &isindirv1alpha3.SopsSecret{}
	TestSecretObject01 := &isindirv1alpha3.SopsSecret{}
	TestSecretObject02 := &isindirv1alpha3.SopsSecret{}
	BeforeEach(func() {
		// 00 secret
		content, err := os.ReadFile(filepath.Join("..", "..", "config", "age-test-key", "00-test-secrets.yaml"))
		Expect(err).Should(BeNil())

		obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(content, nil, nil)
		TestSecretObject00 = obj.(*isindirv1alpha3.SopsSecret)
		Expect(err).Should(BeNil())

		// 01 secret
		content, err = os.ReadFile(filepath.Join("..", "..", "config", "age-test-key", "01-test-secrets.yaml"))
		Expect(err).Should(BeNil())

		obj, _, err = scheme.Codecs.UniversalDeserializer().Decode(content, nil, nil)
		TestSecretObject01 = obj.(*isindirv1alpha3.SopsSecret)
		Expect(err).Should(BeNil())

		// 02 secret
		content, err = os.ReadFile(filepath.Join("..", "..", "config", "age-test-key", "02-test-secrets.yaml"))
		Expect(err).Should(BeNil())

		obj, _, err = scheme.Codecs.UniversalDeserializer().Decode(content, nil, nil)
		TestSecretObject02 = obj.(*isindirv1alpha3.SopsSecret)
		Expect(err).Should(BeNil())
	})

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		SopsSecretName      = "test-sops-secret"
		SopsSecretNamespace = "default"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	// This is to ensure test environment is configured correctly
	Context("When Running controller reconciler", func() {
		It("It should have SOPS env variables defined", func() {
			// Key env variable must be set correctly
			Expect(os.Getenv("SOPS_AGE_RECIPIENTS")).To(Equal("age1pnmp2nq5qx9z4lpmachyn2ld07xjumn98hpeq77e4glddu96zvms9nn7c8"))

			// File containing private key must exist
			ageKeyFileName := os.Getenv("SOPS_AGE_KEY_FILE")
			_, err := os.Stat(ageKeyFileName)
			Expect(err).To(BeNil())
		}, float64(timeout))
	})

	Context("When Creating Malformed SopsSecret Object", func() {
		It("Should Fail to Create SopsSecret", func() {
			By("By creating a new SopsSecret")
			ctx := context.Background()
			sopsSecret := &isindirv1alpha3.SopsSecret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "github.com/isindir/sops-secrets-operator/api/v1alpha3",
					Kind:       "SopsSecret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      SopsSecretName,
					Namespace: SopsSecretNamespace,
				},
				Spec: isindirv1alpha3.SopsSecretSpec{
					Suspend:         true,
					SecretTemplates: []isindirv1alpha3.SopsSecretTemplate{},
				},
			}
			Expect(controller.K8sClient.Create(ctx, sopsSecret)).NotTo(Succeed())
		}, float64(timeout))
	})

	Context("When Creating Correctly Defined SopsSecret Object", func() {
		It("Should Succeed to perform tests using SopsSecret 00", func() {
			By("By creating a new SopsSecret version 00")
			ctx := context.Background()

			Expect(controller.K8sClient.Create(ctx, TestSecretObject00)).To(Succeed())
			time.Sleep(10 * time.Second)

			By("By checking that correct number of secrets was created")
			listCommandOptions := &client.ListOptions{Namespace: "default"}
			secretsList := &corev1.SecretList{}
			Expect(controller.K8sClient.List(ctx, secretsList, listCommandOptions)).To(Succeed())
			// 5 from SopsSecret object + 1 for Service Account
			Expect(len(secretsList.Items)).To(Equal(5))

			By("By checking content of token stringdata test secret")
			testSecret := &corev1.Secret{}
			tagrgetSecretNamespacedName := &types.NamespacedName{Namespace: "default", Name: "test-stringdata-token"}
			Expect(controller.K8sClient.Get(ctx, *tagrgetSecretNamespacedName, testSecret)).To(Succeed())
			Expect(string(testSecret.Data["token"])).To(Equal("Wb4ziZdELkdUf6m6KtNd7iRjjQRvSeJno5meH4NAGHFmpqJyEsekZ2WjX232s4Gj"))

			By("By checking the secret type of test secret without an explicit type")
			Expect(testSecret.Type).To(Equal(corev1.SecretTypeOpaque))

			By("By checking content of token data test secret")
			tagrgetSecretNamespacedName = &types.NamespacedName{Namespace: "default", Name: "test-data-token"}
			Expect(controller.K8sClient.Get(ctx, *tagrgetSecretNamespacedName, testSecret)).To(Succeed())
			Expect(string(testSecret.Data["token"])).To(Equal("Wb4ziZdELkdUf6m6KtNd7iRjjQRvSeJno5meH4NAGHFmpqJyEsekZ2WjX232s4Gj"))

			By("By checking docker secret type")
			tagrgetSecretNamespacedName = &types.NamespacedName{Namespace: "default", Name: "test-type-docker-login"}
			Expect(controller.K8sClient.Get(ctx, *tagrgetSecretNamespacedName, testSecret)).To(Succeed())
			Expect(testSecret.Type).To(Equal(corev1.SecretTypeDockerConfigJson))

			By("By checking custom secret type type")
			tagrgetSecretNamespacedName = &types.NamespacedName{Namespace: "default", Name: "test-type-custom-secret-type"}
			Expect(controller.K8sClient.Get(ctx, *tagrgetSecretNamespacedName, testSecret)).To(Succeed())
			Expect(testSecret.Type).To(Equal(corev1.SecretType("custom/type")))

			By("By checking jenkins test secret contains 1 label and 1 annotation")
			tagrgetSecretNamespacedName = &types.NamespacedName{Namespace: "default", Name: "test-labels-annotations-jenkins-secret"}
			Expect(controller.K8sClient.Get(ctx, *tagrgetSecretNamespacedName, testSecret)).To(Succeed())
			Expect(string(testSecret.Data["username"])).To(Equal("myUsername"))
			Expect(string(testSecret.Data["password"])).To(Equal("Pa58163word"))
			Expect(testSecret.Labels["jenkins.io/credentials-type"]).To(Equal("usernamePassword"))
			Expect(testSecret.Annotations["jenkins.io/credentials-description"]).To(Equal("credentials from Kubernetes"))

			By("By updating a managed k8s secret value outside of SopsSecret object")
			testSecret.Data["username"] = []byte("newUsername")
			Expect(controller.K8sClient.Update(ctx, testSecret)).To(Succeed())
			time.Sleep(10 * time.Second)
			Expect(controller.K8sClient.Get(ctx, *tagrgetSecretNamespacedName, testSecret)).To(Succeed())
			Expect(string(testSecret.Data["username"])).To(Equal("myUsername"))

			By("By deleting data item from a managed k8s secret value outside of SopsSecret object")
			delete(testSecret.Data, "username")
			Expect(controller.K8sClient.Update(ctx, testSecret)).To(Succeed())
			time.Sleep(10 * time.Second)
			Expect(controller.K8sClient.Get(ctx, *tagrgetSecretNamespacedName, testSecret)).To(Succeed())
			Expect(string(testSecret.Data["username"])).To(Equal("myUsername"))

			By("By checking that status of the SopsSecret is Healthy")
			sourceSopsSecret := &isindirv1alpha3.SopsSecret{}
			sourceSopsSecretNamespacedName := &types.NamespacedName{Namespace: "default", Name: "test-sopssecret"}
			Expect(controller.K8sClient.Get(ctx, *sourceSopsSecretNamespacedName, sourceSopsSecret)).To(Succeed())
			Expect(sourceSopsSecret.Status.Message).To(Equal("Healthy"))

			By("By removing secret template from SopsSecret must remove managed k8s secret")
			// Delete template from SopsSecret and update
			// Delete target secret (envtest will not perform garbage collection)
			copy(sourceSopsSecret.Spec.SecretTemplates[0:], sourceSopsSecret.Spec.SecretTemplates[1:])
			sourceSopsSecret.Spec.SecretTemplates = sourceSopsSecret.Spec.SecretTemplates[:len(sourceSopsSecret.Spec.SecretTemplates)-1]
			Expect(controller.K8sClient.Update(ctx, sourceSopsSecret)).To(Succeed())
			testSecret = &corev1.Secret{}
			tagrgetSecretNamespacedName = &types.NamespacedName{Namespace: "default", Name: "test-stringdata-token"}
			Expect(controller.K8sClient.Get(ctx, *tagrgetSecretNamespacedName, testSecret)).To(Succeed())
			Expect(controller.K8sClient.Delete(ctx, testSecret)).To(Succeed())
			time.Sleep(10 * time.Second)
			secretsList = &corev1.SecretList{}
			Expect(controller.K8sClient.List(ctx, secretsList, listCommandOptions)).To(Succeed())

			// 4 from SopsSecret object + 1 for Service Account
			Expect(len(secretsList.Items)).To(Equal(4))
			Expect(controller.K8sClient.Get(ctx, *sourceSopsSecretNamespacedName, sourceSopsSecret)).To(Succeed())
			Expect(sourceSopsSecret.Status.Message).To(Equal("Healthy"))

			By("By deleting SopsSecret version 00")
			Expect(controller.K8sClient.Delete(ctx, TestSecretObject00)).To(Succeed())
		}, float64(timeout))
	})

	Context("When Creating Syntactically Correct SopsSecret Object with broken encrypted data", func() {
		It("Should Succeed to Create SopsSecret 01", func() {
			By("By creating a new SopsSecret version 01")
			ctx := context.Background()
			Expect(controller.K8sClient.Create(ctx, TestSecretObject01)).To(Succeed())
			time.Sleep(10 * time.Second)

			By("By checking that status of the SopsSecret is 'Decryption error'")
			sourceSopsSecret := &isindirv1alpha3.SopsSecret{}
			sourceSopsSecretNamespacedName := &types.NamespacedName{Namespace: "default", Name: "test-sopssecret-01"}
			Expect(controller.K8sClient.Get(ctx, *sourceSopsSecretNamespacedName, sourceSopsSecret)).To(Succeed())
			Expect(sourceSopsSecret.Status.Message).To(Equal("Decryption error"))

			By("By deleting SopsSecret version 01")
			Expect(controller.K8sClient.Delete(ctx, TestSecretObject01)).To(Succeed())
		})
	})

	Context("When Creating Correctly Defined SopsSecret Object when pre-existing not owned secret blocks child creation", func() {
		It("Should Succeed to run tests using SopsSecret 02", func() {
			By("By creating a new not owned by controller plain kubernetes secret 'not-owned-secret-02'")
			ctx := context.Background()
			testSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "not-owned-secret-02",
				},
				Type: corev1.SecretTypeOpaque,
			}
			Expect(controller.K8sClient.Create(ctx, testSecret)).To(Succeed())
			time.Sleep(10 * time.Second)

			By("By creating a new SopsSecret version 02")
			Expect(controller.K8sClient.Create(ctx, TestSecretObject02)).To(Succeed())
			time.Sleep(10 * time.Second)

			By("By checking that status of the SopsSecret is 'Child secret is not owned by controller error'")
			sourceSopsSecret := &isindirv1alpha3.SopsSecret{}
			sourceSopsSecretNamespacedName := &types.NamespacedName{Namespace: "default", Name: "test-sopssecret-02"}
			Expect(controller.K8sClient.Get(ctx, *sourceSopsSecretNamespacedName, sourceSopsSecret)).To(Succeed())
			Expect(sourceSopsSecret.Status.Message).To(Equal("Child secret is not owned by controller error"))

			By("By deleting SopsSecret version 02")
			Expect(controller.K8sClient.Delete(ctx, TestSecretObject02)).To(Succeed())
		})
	})

	// TODO: check pre-existing k8s secret being taken over by SopsSecret using sops managed annotation
	// TODO: check that sopssecret is suspended correctly - not processed - "Reconciliation is suspended"
	// TODO: check the error message is "createKubeSecretFromTemplate(): secret template name must be specified and not empty string".
	//       when child secret template name is empty
	// TODO: check all types of secret - BasicAuth, SSHAuth, BootstrapToken, TLS, Dockercfg, ServiceAccountToken???
})

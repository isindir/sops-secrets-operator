package sopssecret

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	isindirv1alpha1 "github.com/isindir/sops-secrets-operator/pkg/apis/isindir/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	// TODO: use to unencrypt sops encrypted data - !!TOP!!
	"go.mozilla.org/sops"
	sopsaes "go.mozilla.org/sops/aes"
	sopsdotenv "go.mozilla.org/sops/stores/dotenv"
	sopsjson "go.mozilla.org/sops/stores/json"
	sopsyaml "go.mozilla.org/sops/stores/yaml"
	//decrypt "github.com/mozilla/sops/decrypt"
	//decrypt "go.mozilla.org/sops/cmd/sops"
)

var log = logf.Log.WithName("controller_sopssecret")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new SopsSecret Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileSopsSecret{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("sopssecret-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource SopsSecret
	err = c.Watch(&source.Kind{Type: &isindirv1alpha1.SopsSecret{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Secrets and requeue the owner SopsSecret
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &isindirv1alpha1.SopsSecret{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileSopsSecret implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileSopsSecret{}

// ReconcileSopsSecret reconciles a SopsSecret object
type ReconcileSopsSecret struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a SopsSecret object and
// makes changes based on the state read and what is in the SopsSecret.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileSopsSecret) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	reqLogger := log.WithValues(
		"Request.Namespace",
		request.Namespace,
		"Request.Name",
		request.Name,
	)
	reqLogger.Info("Reconciling SopsSecret")

	// Fetch the SopsSecret Encrypted instance
	instanceEncrypted := &isindirv1alpha1.SopsSecret{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instanceEncrypted)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("Request object not found, could have been deleted after reconcile request.")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Info("Error reading the object - requeue the request.")
		return reconcile.Result{}, err
	}

	// TODO: decrypt data_template, if fails - set to reconcile on the next loop and log
	//reqBodyBytes := new(bytes.Buffer)
	instance := &isindirv1alpha1.SopsSecret{}
	//err = json.NewEncoder(reqBodyBytes).Encode(instanceEncrypted)
	reqBodyBytes, err := json.Marshal(instanceEncrypted)
	if err != nil {
		reqLogger.Info("Failed to convert encrypted sops secret to bytes[].")
		return reconcile.Result{}, err
	}

	//decryptedInstanceBytes, err := decrypt.Data(reqBodyBytes, "json")
	decryptedInstanceBytes, err := customDecryptData(reqBodyBytes, "json")
	if err != nil {
		reqLogger.Info("Failed to Decrypt encrypted sops secret instance.")
		return reconcile.Result{}, err
	}

	// Decrypted instance is empty sturcture here
	err = json.Unmarshal(decryptedInstanceBytes, &instance)
	if err != nil {
		reqLogger.Info("Failed to Unmarshal decrypted sops secret instance.")
		return reconcile.Result{}, err
	}

	// TODO: from here to bottom check all usage of instance vs instanceEncrypted

	// Garbage collection logic - for templates removed from SopsSecret
	// Get List of all kube secrets for this sops secret in this namespace.
	existingSecretList := &corev1.SecretList{}
	labelSelector := labels.SelectorFromSet(labelsForSecret(instanceEncrypted))
	listOps := &client.ListOptions{
		Namespace:     request.Namespace,
		LabelSelector: labelSelector,
	}
	// Obtain List of secrets - filter by labels
	if err = r.client.List(
		context.TODO(),
		listOps,
		existingSecretList,
	); err != nil {
		return reconcile.Result{}, err
	}

	// Garbage collection loop - iterate through all fetched secrets and check
	// for matching any template by name, if not - delete secret
	for _, existingKubeSecret := range existingSecretList.Items {
		found := false
		for _, secretTemplateValue := range instance.Spec.SecretsTemplate {
			if secretTemplateValue.Name == existingKubeSecret.Name {
				found = true
				break
			}
		}
		if !found {
			err = r.client.Delete(
				context.TODO(),
				&existingKubeSecret,
				client.GracePeriodSeconds(0),
			)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	reqLogger.Info("Enetring template data loop.")
	for _, secretTemplateValue := range instance.Spec.SecretsTemplate {
		// Define a new secret object
		secret := newSecretForCR(instance, &secretTemplateValue)

		// Set SopsSecret instance as the owner and controller
		if err := controllerutil.SetControllerReference(
			instance,
			secret,
			r.scheme,
		); err != nil {
			return reconcile.Result{}, err
		}

		// Check if this Secret already exists
		foundSecret := &corev1.Secret{}
		err = r.client.Get(
			context.TODO(),
			types.NamespacedName{
				Name:      secret.Name,
				Namespace: secret.Namespace,
			},
			foundSecret,
		)
		if err != nil && errors.IsNotFound(err) {
			reqLogger.Info(
				"Creating a new Secret",
				"Secret.Namespace",
				secret.Namespace,
				"Secret.Name",
				secret.Name,
			)
			err = r.client.Create(context.TODO(), secret)
			if err != nil {
				return reconcile.Result{}, err
			}

			// Secret created successfully - don't requeue
			return reconcile.Result{}, nil
		} else if err != nil {
			return reconcile.Result{}, err
		} else {

			// Secret already exists - enforce update
			reqLogger.Info(
				"Secret already exists: Update",
				"Secret.Namespace",
				foundSecret.Namespace,
				"Secret.Name",
				foundSecret.Name,
			)
			foundSecret.Labels = secret.Labels
			foundSecret.Annotations = secret.Annotations
			foundSecret.StringData = secret.StringData

			if err = r.client.Update(context.TODO(), foundSecret); err != nil {
				return reconcile.Result{}, err
			}
		}
	}
	return reconcile.Result{}, nil
}

// newSecretForCR returns a secret with the same namespace as the cr
func newSecretForCR(
	cr *isindirv1alpha1.SopsSecret,
	secretTpl *isindirv1alpha1.SopsSecretTemplate,
) *corev1.Secret {

	// Construct labels for the secret
	// TODO: instead of using label for GC - find the way to query secrets by
	// parent, than this label is not needed anymore
	// see: https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/
	labels := labelsForSecret(cr)
	for key, value := range secretTpl.Labels {
		labels[key] = value
	}

	// Construct annotations for the secret
	annotations := make(map[string]string)
	for key, value := range secretTpl.Annotations {
		annotations[key] = value
	}

	// Construct Data for the secret
	data := make(map[string]string)
	for key, value := range secretTpl.Data {
		data[key] = value
	}

	reqLogger := log.WithValues(
		"Request.Namespace",
		cr.Namespace,
		"Request.Name",
		cr.Name,
	)
	reqLogger.Info(fmt.Sprintf(
		"Processing secret %s.%s.%s %s:%s",
		cr.Kind,
		cr.APIVersion,
		cr.Name,
		cr.Namespace,
		secretTpl.Name,
	))

	// return resulting secret
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        secretTpl.Name,
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: data,
	}
}

func sanitizeLabel(str string) string {
	var replacer = strings.NewReplacer("/", ".")
	return replacer.Replace(str)
}

func labelsForSecret(cr *isindirv1alpha1.SopsSecret) map[string]string {
	return map[string]string{
		"sopssecret": sanitizeLabel(
			fmt.Sprintf("%s.%s.%s", cr.Kind, cr.APIVersion, cr.Name),
		),
	}
}

// Data is a helper that takes encrypted data and a format string,
// decrypts the data and returns its cleartext in an []byte.
// The format string can be `json`, `yaml`, `dotenv` or `binary`.
// If the format string is empty, binary format is assumed.
func customDecryptData(data []byte, format string) (cleartext []byte, err error) {
	// Initialize a Sops JSON store
	var store sops.Store
	switch format {
	case "json":
		store = &sopsjson.Store{}
	case "yaml":
		store = &sopsyaml.Store{}
	case "dotenv":
		store = &sopsdotenv.Store{}
	default:
		store = &sopsjson.BinaryStore{}
	}
	// Load SOPS file and access the data key
	tree, err := store.LoadEncryptedFile(data)
	if err != nil {
		return nil, err
	}
	key, err := tree.Metadata.GetDataKey()
	if err != nil {
		return nil, err
	}

	// Decrypt the tree
	cipher := sopsaes.NewCipher()
	_, err = tree.Decrypt(key, cipher)
	if err != nil {
		return nil, err
	}

	return store.EmitPlainFile(tree.Branches)
}

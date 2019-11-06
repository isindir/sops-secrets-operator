package sopssecret

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	isindirv1alpha1 "github.com/isindir/sops-secrets-operator/pkg/apis/isindir/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	"go.mozilla.org/sops"
	sopsaes "go.mozilla.org/sops/aes"
	sopsdotenv "go.mozilla.org/sops/stores/dotenv"
	sopsjson "go.mozilla.org/sops/stores/json"
	sopsyaml "go.mozilla.org/sops/stores/yaml"
)

var log = logf.Log.WithName("controller_sopssecret")

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

	instance, err := decryptSopsSecretInstance(instanceEncrypted, reqLogger)
	if err != nil {
		reqLogger.Info("Decryption error.")
		return reconcile.Result{}, err
	}

	// Garbage collection logic - using the fact that owned objects automatically get cleaned up by k8s

	reqLogger.Info("Enetring template data loop.")
	for _, secretTemplateValue := range instance.Spec.SecretsTemplate {
		// Define a new secret object
		newSecret, err := newSecretForCR(instance, &secretTemplateValue)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Set SopsSecret instance as the owner and controller
		if err := controllerutil.SetControllerReference(
			instance,
			newSecret,
			r.scheme,
		); err != nil {
			return reconcile.Result{}, err
		}

		// Check if this Secret already exists
		foundSecret := &corev1.Secret{}
		err = r.client.Get(
			context.TODO(),
			types.NamespacedName{
				Name:      newSecret.Name,
				Namespace: newSecret.Namespace,
			},
			foundSecret,
		)
		if errors.IsNotFound(err) {
			reqLogger.Info(
				"Creating a new Secret",
				"Secret.Namespace",
				newSecret.Namespace,
				"Secret.Name",
				newSecret.Name,
			)
			err = r.client.Create(context.TODO(), newSecret)
			foundSecret = newSecret
		}
		if err != nil {
			return reconcile.Result{}, err
		}

		if !metav1.IsControlledBy(foundSecret, instance) {
			return reconcile.Result{}, fmt.Errorf("Secret isn't currently owned by sops-secrets-operator")
		}

		origSecret := foundSecret
		foundSecret = foundSecret.DeepCopy()

		foundSecret.Data = newSecret.Data
		foundSecret.Type = newSecret.Type
		foundSecret.ObjectMeta.Annotations = newSecret.ObjectMeta.Annotations
		foundSecret.ObjectMeta.Labels = newSecret.ObjectMeta.Labels

		reqLogger.Info(
			"todo rm - Secret already exists checking for updates",
			"Secret.Namespace",
			foundSecret.Namespace,
			"Secret.Name",
			foundSecret.Name,
		)

		if !apiequality.Semantic.DeepEqual(origSecret, foundSecret) {
			reqLogger.Info(
				"Secret already exists and needs updated",
				"Secret.Namespace",
				foundSecret.Namespace,
				"Secret.Name",
				foundSecret.Name,
			)
			if err = r.client.Update(context.TODO(), foundSecret); err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	return reconcile.Result{}, nil
}

// decryptSopsSecretInstance decrypts data_template
func decryptSopsSecretInstance(
	instanceEncrypted *isindirv1alpha1.SopsSecret,
	reqLogger logr.Logger,
) (*isindirv1alpha1.SopsSecret, error) {
	instance := &isindirv1alpha1.SopsSecret{}
	reqBodyBytes, err := json.Marshal(instanceEncrypted)
	if err != nil {
		reqLogger.Info("Failed to convert encrypted sops secret to bytes[].")
		return nil, err
	}

	decryptedInstanceBytes, err := customDecryptData(reqBodyBytes, "json")
	if err != nil {
		reqLogger.Info("Failed to Decrypt encrypted sops secret instance.")
		return nil, err
	}

	// Decrypted instance is empty structure here
	err = json.Unmarshal(decryptedInstanceBytes, &instance)
	if err != nil {
		reqLogger.Info("Failed to Unmarshal decrypted sops secret instance.")
		return nil, err
	}

	return instance, nil
}

// newSecretForCR returns a secret with the same namespace as the cr
func newSecretForCR(
	cr *isindirv1alpha1.SopsSecret,
	secretTpl *isindirv1alpha1.SopsSecretTemplate,
) (*corev1.Secret, error) {
	labels := make(map[string]string)
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

	if secretTpl.Name == "" {
		return nil, fmt.Errorf("newSecretForCR(): secret template name must be specified and not empty string")
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
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        secretTpl.Name,
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: data,
	}
	return secret, nil
}

func sanitizeLabel(str string) string {
	var replacer = strings.NewReplacer("/", ".")
	return replacer.Replace(str)
}

// Data is a helper that takes encrypted data and a format string,
// decrypts the data and returns its cleartext in an []byte.
// The format string can be `json`, `yaml`, `dotenv` or `binary`.
// If the format string is empty, binary format is assumed.
// NOTE: this function is taken from sops code and adjusted
//       to ignore mac, as CR will always be mutated in k8s
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

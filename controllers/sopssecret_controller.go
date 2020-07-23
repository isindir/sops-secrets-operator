package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	isindirv1alpha2 "github.com/isindir/sops-secrets-operator/api/v1alpha2"

	"go.mozilla.org/sops/v3"
	sopsaes "go.mozilla.org/sops/v3/aes"
	sopsdotenv "go.mozilla.org/sops/v3/stores/dotenv"
	sopsjson "go.mozilla.org/sops/v3/stores/json"
	sopsyaml "go.mozilla.org/sops/v3/stores/yaml"
)

// SopsSecretReconciler reconciles a SopsSecret object
type SopsSecretReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=isindir.github.com,resources=sopssecrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=isindir.github.com,resources=sopssecrets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs="*"
// +kubebuilder:rbac:groups="",resources=secrets/status,verbs=get;update;patch

func (r *SopsSecretReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("sopssecret", req.NamespacedName)

	// my logic here
	r.Log.Info("Reconciling SopsSecret")

	instanceEncrypted := &isindirv1alpha2.SopsSecret{}
	err := r.Get(context.TODO(), req.NamespacedName, instanceEncrypted)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			r.Log.Info("Request object not found, could have been deleted after reconcile request.")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		r.Log.Info("Error reading the object - requeue the request.")
		return reconcile.Result{}, err
	}

	instance, err := decryptSopsSecretInstance(instanceEncrypted, r.Log)
	if err != nil {
		r.Log.Info("Decryption error.")
		return reconcile.Result{}, err
	}

	// Garbage collection logic - using the fact that owned objects automatically get cleaned up by k8s

	r.Log.Info("Enetring template data loop.")
	for _, secretTemplateValue := range instance.Spec.SecretsTemplate {
		// Define a new secret object
		newSecret, err := newSecretForCR(instance, &secretTemplateValue, r.Log)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Set SopsSecret instance as the owner and controller
		if err := controllerutil.SetControllerReference(
			instance,
			newSecret,
			r.Scheme,
		); err != nil {
			return reconcile.Result{}, err
		}

		// Check if this Secret already exists
		foundSecret := &corev1.Secret{}
		err = r.Get(
			context.TODO(),
			types.NamespacedName{
				Name:      newSecret.Name,
				Namespace: newSecret.Namespace,
			},
			foundSecret,
		)
		if errors.IsNotFound(err) {
			r.Log.Info("Creating a new Secret")
			err = r.Create(context.TODO(), newSecret)
			foundSecret = newSecret.DeepCopy()
		}
		if err != nil {
			return reconcile.Result{}, err
		}

		if !metav1.IsControlledBy(foundSecret, instance) {
			return reconcile.Result{}, fmt.Errorf("secret isn't currently owned by sops-secrets-operator")
		}

		origSecret := foundSecret
		foundSecret = foundSecret.DeepCopy()

		foundSecret.StringData = newSecret.StringData
		foundSecret.Type = newSecret.Type
		foundSecret.ObjectMeta.Annotations = newSecret.ObjectMeta.Annotations
		foundSecret.ObjectMeta.Labels = newSecret.ObjectMeta.Labels

		if !apiequality.Semantic.DeepEqual(origSecret, foundSecret) {
			r.Log.Info("Secret already exists and needs updated")
			if err = r.Update(context.TODO(), foundSecret); err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *SopsSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&isindirv1alpha2.SopsSecret{}).
		Complete(r)
}

// newSecretForCR returns a secret with the same namespace as the cr
func newSecretForCR(
	cr *isindirv1alpha2.SopsSecret,
	secretTpl *isindirv1alpha2.SopsSecretTemplate,
	reqLogger logr.Logger,
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

	reqLogger.Info(fmt.Sprintf(
		"Processing secret %s.%s.%s.%s %s:%s",
		cr.Kind,
		cr.APIVersion,
		cr.Name,
		secretTpl.Type,
		cr.Namespace,
		secretTpl.Name,
	))

	kubeSecretType := getSecretType(secretTpl.Type)

	// return resulting secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        secretTpl.Name,
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Type:       kubeSecretType,
		StringData: data,
	}
	return secret, nil
}

func getSecretType(paramType string) corev1.SecretType {
	// by default secret type is Opaque
	kubeSecretType := corev1.SecretTypeOpaque
	if paramType == "kubernetes.io/service-account-token" {
		kubeSecretType = corev1.SecretTypeServiceAccountToken
	}
	if paramType == "kubernetes.io/dockercfg" {
		kubeSecretType = corev1.SecretTypeDockercfg
	}
	if paramType == "kubernetes.io/dockerconfigjson" {
		kubeSecretType = corev1.SecretTypeDockerConfigJson
	}
	if paramType == "kubernetes.io/basic-auth" {
		kubeSecretType = corev1.SecretTypeBasicAuth
	}
	if paramType == "kubernetes.io/ssh-auth" {
		kubeSecretType = corev1.SecretTypeSSHAuth
	}
	if paramType == "kubernetes.io/tls" {
		kubeSecretType = corev1.SecretTypeTLS
	}
	if paramType == "bootstrap.kubernetes.io/token" {
		kubeSecretType = corev1.SecretTypeBootstrapToken
	}
	return kubeSecretType
}

// decryptSopsSecretInstance decrypts data_template
func decryptSopsSecretInstance(
	instanceEncrypted *isindirv1alpha2.SopsSecret,
	reqLogger logr.Logger,
) (*isindirv1alpha2.SopsSecret, error) {
	instance := &isindirv1alpha2.SopsSecret{}
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

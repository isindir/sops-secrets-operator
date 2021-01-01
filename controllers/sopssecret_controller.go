package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
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
	sopslogging "go.mozilla.org/sops/v3/logging"
	sopsdotenv "go.mozilla.org/sops/v3/stores/dotenv"
	sopsjson "go.mozilla.org/sops/v3/stores/json"
	sopsyaml "go.mozilla.org/sops/v3/stores/yaml"
)

// SopsSecretReconciler reconciles a SopsSecret object
type SopsSecretReconciler struct {
	client.Client
	Log          logr.Logger
	Scheme       *runtime.Scheme
	RequeueAfter int64
}

// Reconcile - main reconcile loop of the controller
// +kubebuilder:rbac:groups=isindir.github.com,resources=sopssecrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=isindir.github.com,resources=sopssecrets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs="*"
// +kubebuilder:rbac:groups="",resources=secrets/status,verbs=get;update;patch
func (r *SopsSecretReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("sopssecret", req.NamespacedName)

	r.Log.Info("Reconciling", "sopssecret", req.NamespacedName)

	instanceEncrypted := &isindirv1alpha2.SopsSecret{}
	err := r.Get(context.TODO(), req.NamespacedName, instanceEncrypted)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			r.Log.Info(
				"Request object not found, could have been deleted after reconcile request",
				"sopssecret",
				req.NamespacedName,
			)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		r.Log.Info(
			"Error reading the object - requeue the request",
			"sopssecret",
			req.NamespacedName,
		)
		return reconcile.Result{}, err
	}

	instance, err := decryptSopsSecretInstance(instanceEncrypted, r.Log)
	if err != nil {
		//instance.Status.SecretsTotal = len(instance.Spec.SecretsTemplate)
		instanceEncrypted.Status.Message = "Decryption error"

		// will not process instance error as we are already in error mode here
		r.Status().Update(context.Background(), instanceEncrypted)

		// Failed to decrypt, re-schedule reconciliation in 5 minutes
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(r.RequeueAfter) * time.Minute}, nil
	}

	// iterating over secret templates
	r.Log.Info("Entering template data loop", "sopssecret", req.NamespacedName)
	for _, secretTemplateValue := range instance.Spec.SecretsTemplate {
		// Define a new secret object
		newSecret, err := newSecretForCR(instance, &secretTemplateValue, r.Log)
		if err != nil {
			instanceEncrypted.Status.Message = "New child secret creation error"
			r.Status().Update(context.Background(), instanceEncrypted)

			r.Log.Info(
				"New child secret creation error",
				"sopssecret",
				req.NamespacedName,
				"error",
				err,
			)
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(r.RequeueAfter) * time.Minute}, nil
		}

		// Set SopsSecret instance as the owner and controller
		if err := controllerutil.SetControllerReference(
			instance,
			newSecret,
			r.Scheme,
		); err != nil {
			instanceEncrypted.Status.Message = "Setting controller ownership of the child secret error"
			r.Status().Update(context.Background(), instanceEncrypted)

			r.Log.Info(
				"Setting controller ownership of the child secret error",
				"sopssecret",
				req.NamespacedName,
				"error",
				err,
			)
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(r.RequeueAfter) * time.Minute}, nil
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
			r.Log.Info(
				"Creating a new Secret",
				"sopssecret",
				req.NamespacedName,
				"message",
				err,
			)
			err = r.Create(context.TODO(), newSecret)
			foundSecret = newSecret.DeepCopy()
		}
		if err != nil {
			instanceEncrypted.Status.Message = "Unknown Error"
			r.Status().Update(context.Background(), instanceEncrypted)

			r.Log.Info(
				"Unknown Error",
				"sopssecret",
				req.NamespacedName,
				"error",
				err,
			)
			return reconcile.Result{}, nil
		}

		if !metav1.IsControlledBy(foundSecret, instance) {
			instanceEncrypted.Status.Message = "Child secret is not owned by controller error"
			r.Status().Update(context.Background(), instanceEncrypted)

			r.Log.Info(
				"Child secret is not owned by controller or sopssecret Error",
				"sopssecret",
				req.NamespacedName,
				"error",
				fmt.Errorf("sopssecret has a conflict with existing kubernetes secret resource, potential reasons: target secret already pre-existed or is managed by multiple sops secrets"),
			)
			return reconcile.Result{}, nil
		}

		origSecret := foundSecret
		foundSecret = foundSecret.DeepCopy()

		foundSecret.StringData = newSecret.StringData
		foundSecret.Data = map[string][]byte{}
		foundSecret.Type = newSecret.Type
		foundSecret.ObjectMeta.Annotations = newSecret.ObjectMeta.Annotations
		foundSecret.ObjectMeta.Labels = newSecret.ObjectMeta.Labels

		if !apiequality.Semantic.DeepEqual(origSecret, foundSecret) {
			r.Log.Info(
				"Secret already exists and needs to be refreshed",
				"secret",
				foundSecret.Name,
				"namespace",
				foundSecret.Namespace,
			)
			if err = r.Update(context.TODO(), foundSecret); err != nil {
				instanceEncrypted.Status.Message = "Child secret update error"
				r.Status().Update(context.Background(), instanceEncrypted)

				r.Log.Info(
					"Child secret update error",
					"sopssecret",
					req.NamespacedName,
					"error",
					err,
				)
				return reconcile.Result{}, nil
			}
			r.Log.Info(
				"Secret successfully refreshed",
				"secret",
				foundSecret.Name,
				"namespace",
				foundSecret.Namespace,
			)
		}
	}

	instanceEncrypted.Status.Message = "Healthy"
	r.Status().Update(context.Background(), instanceEncrypted)

	r.Log.Info(
		"SopsSecret is Healthy",
		"sopssecret",
		req.NamespacedName,
	)
	return ctrl.Result{}, nil
}

// SetupWithManager - setup with manager
func (r *SopsSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// Set logging level
	sopslogging.SetLevel(logrus.InfoLevel)

	// Set logrus logs to be discarded
	for k := range sopslogging.Loggers {
		sopslogging.Loggers[k].Out = ioutil.Discard
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&isindirv1alpha2.SopsSecret{}).
		Owns(&corev1.Secret{}).
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

	reqLogger.Info("Processing", "sopssecret",
		fmt.Sprintf(
			"%s.%s.%s",
			cr.Kind,
			cr.APIVersion,
			cr.Name,
		),
		"type",
		secretTpl.Type,
		"namespace", cr.Namespace,
		"templateItem",
		fmt.Sprintf("secret/%s", secretTpl.Name),
	)

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

// decryptSopsSecretInstance decrypts spec.secretTemplates
func decryptSopsSecretInstance(
	instanceEncrypted *isindirv1alpha2.SopsSecret,
	reqLogger logr.Logger,
) (*isindirv1alpha2.SopsSecret, error) {
	instance := &isindirv1alpha2.SopsSecret{}
	reqBodyBytes, err := json.Marshal(instanceEncrypted)
	if err != nil {
		reqLogger.Info(
			"Failed to convert encrypted sops secret to bytes[]",
			"sopssecret",
			fmt.Sprintf("%s/%s", instanceEncrypted.Namespace, instanceEncrypted.Name),
			"error",
			err,
		)
		return nil, err
	}

	decryptedInstanceBytes, err := customDecryptData(reqBodyBytes, "json")
	if err != nil {
		reqLogger.Info(
			"Failed to Decrypt encrypted sops secret instance",
			"sopssecret",
			fmt.Sprintf("%s/%s", instanceEncrypted.Namespace, instanceEncrypted.Name),
			"error",
			err,
		)
		return nil, err
	}

	// Decrypted instance is empty structure here
	err = json.Unmarshal(decryptedInstanceBytes, &instance)
	if err != nil {
		reqLogger.Info(
			"Failed to Unmarshal decrypted sops secret instance",
			"sopssecret",
			fmt.Sprintf("%s/%s", instanceEncrypted.Namespace, instanceEncrypted.Name),
			"error",
			err,
		)
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
	if userErr, ok := err.(sops.UserError); ok {
		err = fmt.Errorf(userErr.UserError())
	}
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

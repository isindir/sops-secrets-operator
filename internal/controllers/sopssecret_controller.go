/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

package controllers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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

	isindirv1alpha3 "github.com/isindir/sops-secrets-operator/api/v1alpha3"

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

//+kubebuilder:rbac:groups=isindir.github.com,resources=sopssecrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=isindir.github.com,resources=sopssecrets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=isindir.github.com,resources=sopssecrets/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs="*"
//+kubebuilder:rbac:groups="",resources=secrets/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// UPDATE-HERE
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.6/pkg/reconcile
func (r *SopsSecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("sopssecret", req.NamespacedName)

	r.Log.V(0).Info("Reconciling", "sopssecret", req.NamespacedName)

	encryptedSopsSecret, finishReconcileLoop, err := r.getEncryptedSopsSecret(ctx, req)
	if finishReconcileLoop {
		return reconcile.Result{}, err
	}

	if r.isSecretSuspended(encryptedSopsSecret, req) {
		sopsSecretsReconciliationsSuspended.Inc()
		return reconcile.Result{}, nil
	}

	plainTextSopsSecret, rescheduleReconcileLoop := r.decryptSopsSecret(encryptedSopsSecret)
	if rescheduleReconcileLoop {
		sopsSecretsReconciliationFailures.Inc()
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(r.RequeueAfter) * time.Minute}, nil
	}

	// Iterate over secret templates
	r.Log.V(1).Info("Entering template data loop", "sopssecret", req.NamespacedName)
	for _, secretTemplate := range plainTextSopsSecret.Spec.SecretsTemplate {

		kubeSecretFromTemplate, rescheduleReconcileLoop := r.newKubeSecretFromTemplate(req, encryptedSopsSecret, plainTextSopsSecret, &secretTemplate)
		if rescheduleReconcileLoop {
			sopsSecretsReconciliationFailures.Inc()
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(r.RequeueAfter) * time.Minute}, nil
		}

		kubeSecretInCluster, rescheduleReconcileLoop := r.getSecretFromClusterOrCreateFromTemplate(ctx, req, encryptedSopsSecret, kubeSecretFromTemplate)
		if rescheduleReconcileLoop {
			sopsSecretsReconciliationFailures.Inc()
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(r.RequeueAfter) * time.Minute}, nil
		}

		rescheduleReconcileLoop = r.isKubeSecretManagedOrAnnotatedToBeManaged(req, encryptedSopsSecret, kubeSecretInCluster)
		if rescheduleReconcileLoop {
			sopsSecretsReconciliationFailures.Inc()
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(r.RequeueAfter) * time.Minute}, nil
		}

		rescheduleReconcileLoop = r.refreshKubeSecretIfNeeded(ctx, req, encryptedSopsSecret, kubeSecretFromTemplate, kubeSecretInCluster)
		if rescheduleReconcileLoop {
			sopsSecretsReconciliationFailures.Inc()
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(r.RequeueAfter) * time.Minute}, nil
		}
	}

	if encryptedSopsSecret.Status.Message != "Healthy" {
		encryptedSopsSecret.Status.Message = "Healthy"
		_ = r.Status().Update(context.Background(), encryptedSopsSecret)
	}
	sopsSecretsReconciliations.Inc()

	r.Log.V(1).Info("SopsSecret is Healthy", "sopssecret", req.NamespacedName)
	return ctrl.Result{}, nil
}

func (r *SopsSecretReconciler) decryptSopsSecret(
	encryptedSopsSecret *isindirv1alpha3.SopsSecret,
) (*isindirv1alpha3.SopsSecret, bool) {
	decryptedSopsSecret, err := decryptSopsSecretInstance(encryptedSopsSecret, r.Log)
	if err != nil {
		encryptedSopsSecret.Status.Message = "Decryption error"

		// will not process plainTextSopsSecret error as we are already in error mode here
		_ = r.Status().Update(context.Background(), encryptedSopsSecret)
		// Failed to decrypt, re-schedule reconciliation in 5 minutes
		return nil, true
	}
	return decryptedSopsSecret, false
}

func (r *SopsSecretReconciler) isKubeSecretManagedOrAnnotatedToBeManaged(
	req ctrl.Request,
	encryptedSopsSecret *isindirv1alpha3.SopsSecret,
	kubeSecretInCluster *corev1.Secret,
) bool {
	// kubeSecretFromTemplate found - perform ownership check
	if !metav1.IsControlledBy(kubeSecretInCluster, encryptedSopsSecret) && !isAnnotatedToBeManaged(kubeSecretInCluster) {
		encryptedSopsSecret.Status.Message = "Child secret is not owned by controller error"
		_ = r.Status().Update(context.Background(), encryptedSopsSecret)

		r.Log.V(0).Info(
			"Child secret is not owned by controller or sopssecret Error",
			"sopssecret", req.NamespacedName,
			"error", fmt.Errorf("sopssecret has a conflict with existing kubernetes secret resource, potential reasons: target secret already pre-existed or is managed by multiple sops secrets"),
		)
		return true
	}
	return false
}

func (r *SopsSecretReconciler) refreshKubeSecretIfNeeded(
	ctx context.Context,
	req ctrl.Request,
	encryptedSopsSecret *isindirv1alpha3.SopsSecret,
	kubeSecretFromTemplate *corev1.Secret,
	kubeSecretInCluster *corev1.Secret,
) bool {
	copyOfKubeSecretInCluster := kubeSecretInCluster.DeepCopy()

	copyOfKubeSecretInCluster.StringData = kubeSecretFromTemplate.StringData
	copyOfKubeSecretInCluster.Data = map[string][]byte{}
	copyOfKubeSecretInCluster.Type = kubeSecretFromTemplate.Type
	copyOfKubeSecretInCluster.ObjectMeta.Annotations = kubeSecretFromTemplate.ObjectMeta.Annotations
	copyOfKubeSecretInCluster.ObjectMeta.Labels = kubeSecretFromTemplate.ObjectMeta.Labels

	if isAnnotatedToBeManaged(kubeSecretInCluster) {
		copyOfKubeSecretInCluster.ObjectMeta.OwnerReferences = kubeSecretFromTemplate.ObjectMeta.OwnerReferences
	}

	if !apiequality.Semantic.DeepEqual(kubeSecretInCluster, copyOfKubeSecretInCluster) {
		r.Log.V(0).Info(
			"Secret already exists and needs to be refreshed",
			"secret", copyOfKubeSecretInCluster.Name,
			"namespace", copyOfKubeSecretInCluster.Namespace,
		)
		if err := r.Update(ctx, copyOfKubeSecretInCluster); err != nil {
			encryptedSopsSecret.Status.Message = "Child secret update error"
			_ = r.Status().Update(context.Background(), encryptedSopsSecret)

			r.Log.V(0).Info(
				"Child secret update error",
				"sopssecret", req.NamespacedName,
				"error", err,
			)
			return true
		}
		r.Log.V(0).Info(
			"Secret successfully refreshed",
			"secret", copyOfKubeSecretInCluster.Name,
			"namespace", copyOfKubeSecretInCluster.Namespace,
		)
	}
	return false
}

func (r *SopsSecretReconciler) getSecretFromClusterOrCreateFromTemplate(
	ctx context.Context,
	req ctrl.Request,
	encryptedSopsSecret *isindirv1alpha3.SopsSecret,
	kubeSecretFromTemplate *corev1.Secret,
) (*corev1.Secret, bool) {

	// Check if kubeSecretFromTemplate already exists in the cluster store
	kubeSecretToFindAndCompare := &corev1.Secret{}
	err := r.Get(
		ctx,
		types.NamespacedName{
			Name:      kubeSecretFromTemplate.Name,
			Namespace: kubeSecretFromTemplate.Namespace,
		},
		kubeSecretToFindAndCompare,
	)

	// No kubeSecretFromTemplate alike found - CREATE one
	if errors.IsNotFound(err) {
		r.Log.V(1).Info(
			"Creating a new Secret",
			"sopssecret", req.NamespacedName,
			"message", err,
		)
		err = r.Create(ctx, kubeSecretFromTemplate)
		kubeSecretToFindAndCompare = kubeSecretFromTemplate.DeepCopy()
	}

	// Unknown error while trying to find kubeSecretFromTemplate in cluster - reschedule reconciliation
	if err != nil {
		encryptedSopsSecret.Status.Message = "Unknown Error"
		_ = r.Status().Update(context.Background(), encryptedSopsSecret)

		r.Log.V(0).Info(
			"Unknown Error",
			"sopssecret", req.NamespacedName,
			"error", err,
		)
		return nil, true
	}

	return kubeSecretToFindAndCompare, false
}

func (r *SopsSecretReconciler) newKubeSecretFromTemplate(
	req ctrl.Request,
	encryptedSopsSecret *isindirv1alpha3.SopsSecret,
	plainTextSopsSecret *isindirv1alpha3.SopsSecret,
	secretTemplate *isindirv1alpha3.SopsSecretTemplate,
) (*corev1.Secret, bool) {

	// Define a new secret object
	kubeSecretFromTemplate, err := createKubeSecretFromTemplate(plainTextSopsSecret, secretTemplate, r.Log)
	if err != nil {
		encryptedSopsSecret.Status.Message = "New child secret creation error"
		_ = r.Status().Update(context.Background(), encryptedSopsSecret)

		r.Log.V(0).Info(
			"New child secret creation error",
			"sopssecret", req.NamespacedName,
			"error", err,
		)
		return nil, true
	}

	// Set encryptedSopsSecret as the owner of kubeSecret
	err = controllerutil.SetControllerReference(encryptedSopsSecret, kubeSecretFromTemplate, r.Scheme)
	if err != nil {
		encryptedSopsSecret.Status.Message = "Setting controller ownership of the child secret error"
		_ = r.Status().Update(context.Background(), encryptedSopsSecret)

		r.Log.V(0).Info(
			"Setting controller ownership of the child secret error",
			"sopssecret", req.NamespacedName,
			"error", err,
		)

		return nil, true
	}

	return kubeSecretFromTemplate, false
}

func (r *SopsSecretReconciler) isSecretSuspended(
	encryptedSopsSecret *isindirv1alpha3.SopsSecret, req ctrl.Request) bool {

	// Return early if SopsSecret object is suspended.
	if encryptedSopsSecret.Spec.Suspend {
		r.Log.V(0).Info(
			"Reconciliation is suspended for this object",
			"sopssecret", req.NamespacedName,
		)

		encryptedSopsSecret.Status.Message = "Reconciliation is suspended"
		_ = r.Status().Update(context.Background(), encryptedSopsSecret)

		return true
	}

	return false
}

func (r *SopsSecretReconciler) getEncryptedSopsSecret(
	ctx context.Context, req ctrl.Request) (*isindirv1alpha3.SopsSecret, bool, error) {

	encryptedSopsSecret := &isindirv1alpha3.SopsSecret{}

	err := r.Get(ctx, req.NamespacedName, encryptedSopsSecret)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			r.Log.V(0).Info(
				"Request object not found, could have been deleted after reconcile request",
				"sopssecret",
				req.NamespacedName,
			)
			return nil, true, nil
		}

		// Error reading the object - requeue the request.
		r.Log.V(0).Info(
			"Error reading the object",
			"sopssecret",
			req.NamespacedName,
		)
		return nil, true, err
	}
	return encryptedSopsSecret, false, nil
}

// checks if the annotation equals to "true", and it's case sensitive
func isAnnotatedToBeManaged(secret *corev1.Secret) bool {
	return secret.Annotations[isindirv1alpha3.SopsSecretManagedAnnotation] == "true"
}

// SetupWithManager sets up the controller with the Manager.
func (r *SopsSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// Set logging level
	sopslogging.SetLevel(logrus.InfoLevel)

	// Set logrus logs to be discarded
	for k := range sopslogging.Loggers {
		sopslogging.Loggers[k].Out = io.Discard
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&isindirv1alpha3.SopsSecret{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

// createKubeSecretFromTemplate returns new Kubernetes secret object, created from decrypted SopsSecret Template
func createKubeSecretFromTemplate(
	sopsSecret *isindirv1alpha3.SopsSecret,
	sopsSecretTemplate *isindirv1alpha3.SopsSecretTemplate,
	logger logr.Logger,
) (*corev1.Secret, error) {
	if sopsSecretTemplate.Name == "" {
		return nil, fmt.Errorf("createKubeSecretFromTemplate(): secret template name must be specified and not empty string")
	}

	strData, err := cloneTemplateData(sopsSecretTemplate.StringData, sopsSecretTemplate.Data)
	if err != nil {
		return nil, err
	}

	kubeSecretType := getSecretType(sopsSecretTemplate.Type)
	labels := cloneMap(sopsSecretTemplate.Labels)
	annotations := cloneMap(sopsSecretTemplate.Annotations)

	logger.V(1).Info("Processing",
		"sopssecret", fmt.Sprintf("%s.%s.%s", sopsSecret.Kind, sopsSecret.APIVersion, sopsSecret.Name),
		"type", sopsSecretTemplate.Type,
		"namespace", sopsSecret.Namespace,
		"templateItem", fmt.Sprintf("secret/%s", sopsSecretTemplate.Name),
	)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        sopsSecretTemplate.Name,
			Namespace:   sopsSecret.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Type:       kubeSecretType,
		StringData: strData,
	}
	return secret, nil
}

func cloneMap(oldMap map[string]string) map[string]string {
	newMap := make(map[string]string)

	for key, value := range oldMap {
		newMap[key] = value
	}

	return newMap
}

// add both StringData and Data to strData
func cloneTemplateData(stringData map[string]string, data map[string]string) (map[string]string, error) {
	strData := cloneMap(stringData)
	for key, value := range data {
		decoded, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return nil, fmt.Errorf("createKubeSecretFromTemplate(): data[%v] is not a valid base64 string", key)
		}
		strData[key] = string(decoded)
	}
	return strData, nil
}

func getSecretType(templateSecretType string) corev1.SecretType {
	if templateSecretType == "" {
		return corev1.SecretTypeOpaque
	}
	return corev1.SecretType(templateSecretType)
}

// decryptSopsSecretInstance decrypts spec.secretTemplates
func decryptSopsSecretInstance(
	encryptedSopsSecret *isindirv1alpha3.SopsSecret,
	logger logr.Logger,
) (*isindirv1alpha3.SopsSecret, error) {
	sopsSecretAsBytes, err := json.Marshal(encryptedSopsSecret)
	if err != nil {
		logger.V(0).Info(
			"Failed to convert encrypted sops secret to bytes[]",
			"sopssecret", fmt.Sprintf("%s/%s", encryptedSopsSecret.Namespace, encryptedSopsSecret.Name),
			"error", err,
		)
		return nil, err
	}

	decryptedSopsSecretAsBytes, err := customDecryptData(sopsSecretAsBytes, "json")
	if err != nil {
		logger.V(0).Info(
			"Failed to Decrypt encrypted sops secret decryptedSopsSecret",
			"sopssecret", fmt.Sprintf("%s/%s", encryptedSopsSecret.Namespace, encryptedSopsSecret.Name),
			"error", err,
		)
		return nil, err
	}

	decryptedSopsSecret := &isindirv1alpha3.SopsSecret{}
	err = json.Unmarshal(decryptedSopsSecretAsBytes, &decryptedSopsSecret)
	if err != nil {
		logger.V(0).Info(
			"Failed to Unmarshal decrypted sops secret decryptedSopsSecret",
			"sopssecret", fmt.Sprintf("%s/%s", encryptedSopsSecret.Namespace, encryptedSopsSecret.Name),
			"error", err,
		)
		return nil, err
	}

	return decryptedSopsSecret, nil
}

// Data is a helper that takes encrypted data and a format string,
// decrypts the data and returns its cleartext in an []byte.
// The format string can be `json`, `yaml`, `dotenv` or `binary`.
// If the format string is empty, binary format is assumed.
// NOTE: this function is taken from sops code and adjusted
//
//	to ignore mac, as CR will always be mutated in k8s
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

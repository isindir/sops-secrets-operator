/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

package v1alpha3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// For upstream reference, see https://github.com/mozilla/sops/blob/master/stores/stores.go

const (
	// SopsSecretManagedAnnotation is the name for the annotation for
	// flagging the existing secret be managed by SopsSecret controller.
	SopsSecretManagedAnnotation = "sopssecret/managed"
)

// SopsSecretSpec defines the desired state of SopsSecret
type SopsSecretSpec struct {
	// Secrets template is a list of definitions to create Kubernetes Secrets
	//+kubebuilder:validation:MinItems=1
	//+required
	SecretsTemplate []SopsSecretTemplate `json:"secretTemplates"`

	// This flag tells the controller to suspend the reconciliation of this source.
	//+optional
	Suspend bool `json:"suspend,omitempty"`

	// EnforceNamespace can be used to enforce the creation of the secrets in the same namespace as the SopsSecret resource.
	// Must be used together with Spec.Namespace
	EnforceNamespace bool `json:"enforce_namespace,omitempty"`

	// Namespace can be used to enforce the creation of the secrets in the same namespace as the SopsSecret resource.
	// Must have same value as the SopsSecret resource namespace and EnforceNamespace must be set to true.
	//+optional
	Namespace string `json:"namespace,omitempty"`
}

// SopsSecretTemplate defines the map of secrets to create
type SopsSecretTemplate struct {
	// Name of the Kubernetes secret to create
	//+required
	Name string `json:"name"`

	// Annotations to apply to Kubernetes secret
	//+optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels to apply to Kubernetes secret
	//+optional
	Labels map[string]string `json:"labels,omitempty"`

	// Kubernetes secret type. Default: Opauqe. Possible values: Opauqe,
	// kubernetes.io/service-account-token, kubernetes.io/dockercfg,
	// kubernetes.io/dockerconfigjson, kubernetes.io/basic-auth,
	// kubernetes.io/ssh-auth, kubernetes.io/tls, bootstrap.kubernetes.io/token
	//+optional
	Type string `json:"type,omitempty"`

	// Data map to use in Kubernetes secret (equivalent to Kubernetes Secret object data, please see for more
	// information: https://kubernetes.io/docs/concepts/configuration/secret/#overview-of-secrets)
	//+optional
	Data map[string]string `json:"data,omitempty"`

	// stringData map to use in Kubernetes secret (equivalent to Kubernetes Secret object stringData, please see for more
	// information: https://kubernetes.io/docs/concepts/configuration/secret/#overview-of-secrets)
	//+optional
	StringData map[string]string `json:"stringData,omitempty"`
}

// KmsDataItem defines AWS KMS specific encryption details
type KmsDataItem struct {
	// Arn - KMS key ARN to use
	//+optional
	Arn string `json:"arn,omitempty"`
	// AWS Iam Role
	//+optional
	Role string `json:"role,omitempty"`

	//+optional
	EncryptedKey string `json:"enc,omitempty"`
	// Object creation date
	//+optional
	CreationDate string `json:"created_at,omitempty"`
	//+optional
	AwsProfile string `json:"aws_profile,omitempty"`
}

// PgpDataItem defines PGP specific encryption details
type PgpDataItem struct {
	//+optional
	EncryptedKey string `json:"enc,omitempty"`

	// Object creation date
	//+optional
	CreationDate string `json:"created_at,omitempty"`
	// PGP FingerPrint of the key which can be used for decryption
	//+optional
	FingerPrint string `json:"fp,omitempty"`
}

// AzureKmsItem defines Azure Keyvault Key specific encryption details
type AzureKmsItem struct {
	// Azure KMS vault URL
	//+optional
	VaultURL string `json:"vault_url,omitempty"`
	//+optional
	KeyName string `json:"name,omitempty"`
	//+optional
	Version string `json:"version,omitempty"`
	//+optional
	EncryptedKey string `json:"enc,omitempty"`
	// Object creation date
	//+optional
	CreationDate string `json:"created_at,omitempty"`
}

// AgeItem defines FiloSottile/age specific encryption details
type AgeItem struct {
	// Recipient which private key can be used for decription
	//+optional
	Recipient string `json:"recipient,omitempty"`
	//+optional
	EncryptedKey string `json:"enc,omitempty"`
}

// HcVaultItem defines Hashicorp Vault Key specific encryption details
type HcVaultItem struct {
	//+optional
	VaultAddress string `json:"vault_address,omitempty"`
	//+optional
	EnginePath string `json:"engine_path,omitempty"`
	//+optional
	KeyName string `json:"key_name,omitempty"`
	//+optional
	CreationDate string `json:"created_at,omitempty"`
	//+optional
	EncryptedKey string `json:"enc,omitempty"`
}

// GcpKmsDataItem defines GCP KMS Key specific encryption details
type GcpKmsDataItem struct {
	//+optional
	VaultURL string `json:"resource_id,omitempty"`
	//+optional
	EncryptedKey string `json:"enc,omitempty"`
	// Object creation date
	//+optional
	CreationDate string `json:"created_at,omitempty"`
}

// SopsMetadata defines the encryption details
type SopsMetadata struct {
	// Aws KMS configuration
	//+optional
	AwsKms []KmsDataItem `json:"kms,omitempty"`

	// PGP configuration
	//+optional
	Pgp []PgpDataItem `json:"pgp,omitempty"`

	// Azure KMS configuration
	//+optional
	AzureKms []AzureKmsItem `json:"azure_kv,omitempty"`

	// Hashicorp Vault KMS configurarion
	//+optional
	HcVault []HcVaultItem `json:"hc_vault,omitempty"`

	// Gcp KMS configuration
	//+optional
	GcpKms []GcpKmsDataItem `json:"gcp_kms,omitempty"`

	// Age configuration
	//+optional
	Age []AgeItem `json:"age,omitempty"`

	// Mac - sops setting
	//+optional
	Mac string `json:"mac,omitempty"`

	// LastModified date when SopsSecret was last modified
	//+optional
	LastModified string `json:"lastmodified,omitempty"`

	// Version of the sops tool used to encrypt SopsSecret
	//+optional
	Version string `json:"version,omitempty"`

	// Suffix used to encrypt SopsSecret resource
	//+optional
	EncryptedSuffix string `json:"encrypted_suffix,omitempty"`

	// Regex used to encrypt SopsSecret resource
	// This opstion should be used with more care, as it can make resource unapplicable to the cluster.
	//+optional
	EncryptedRegex string `json:"encrypted_regex,omitempty"`
}

// SopsSecretStatus defines the observed state of SopsSecret
type SopsSecretStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// SopsSecret status message
	//+optional
	Message string `json:"message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// SopsSecret is the Schema for the sopssecrets API
// +kubebuilder:resource:shortName=sops,scope=Namespaced
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.message`
type SopsSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// SopsSecret Spec definition
	Spec SopsSecretSpec `json:"spec,omitempty"`
	// SopsSecret Status information
	Status SopsSecretStatus `json:"status,omitempty"`
	// SopsSecret metadata
	Sops SopsMetadata `json:"sops,omitempty"`
}

//+kubebuilder:object:root=true

// SopsSecretList contains a list of SopsSecret
type SopsSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SopsSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SopsSecret{}, &SopsSecretList{})
}

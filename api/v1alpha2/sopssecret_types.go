package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SopsSecretTemplate defines the map of secrets to create
type SopsSecretTemplate struct {
	// Name is a name of the Kubernetes secret to create
	Name string `json:"name"`

	// Annotations to apply to Kubernetes secret
	// +optional

	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels to apply to Kubernetes secret
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Kubernetes secret type
	// +optional
	Type string `json:"type,omitempty"`

	// Data is data map to use in Kubernetes secret
	Data map[string]string `json:"data"`
}

// SopsSecretSpec defines the desired state of SopsSecret
type SopsSecretSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// SecretsTemplate is a list of secret templates to create Kubernetes Secrets
	// +kubebuilder:validation:MinItems=1
	SecretsTemplate []SopsSecretTemplate `json:"secretTemplates"`
}

// KmsDataItem defines AWS KMS specific encryption details
type KmsDataItem struct {
	// Arn - KMS key ARN to use
	// +optional
	Arn string `json:"arn,omitempty"`

	// +optional
	EncryptedKey string `json:"enc,omitempty"`
	// +optional
	CreationDate string `json:"created_at,omitempty"`
	// +optional
	AwsProfile string `json:"aws_profile,omitempty"`
}

// PgpDataItem defines PGP specific encryption details
type PgpDataItem struct {
	// +optional
	EncryptedKey string `json:"enc,omitempty"`
	// +optional
	CreationDate string `json:"created_at,omitempty"`

	// FingerPrint - PGP FingerPrint to encrypt for
	// +optional
	FingerPrint string `json:"fp,omitempty"`
}

// AzureKmsItem defines Azure Keyvault Key specific encryption details
type AzureKmsItem struct {
	// +optional
	VaultURL string `json:"vault_url,omitempty"`
	// +optional
	KeyName string `json:"name,omitempty"`
	// +optional
	Version string `json:"version,omitempty"`
	// +optional
	EncryptedKey string `json:"enc,omitempty"`
	// +optional
	CreationDate string `json:"created_at,omitempty"`
}

type AgeItem struct {
	// +optional
	Recipient string `json:"recipient,omitempty"`
	// +optional
	EncryptedKey string `json:"enc,omitempty"`
}

// HcVaultItem defines Hashicorp Vault Key specific encryption details
type HcVaultItem struct {
	// +optional
	VaultAddress string `json:"vault_address,omitempty"`
	// +optional
	EnginePath string `json:"engine_path,omitempty"`
	// +optional
	KeyName string `json:"key_name,omitempty"`
	// +optional
	CreationDate string `json:"created_at,omitempty"`
}

// GcpKmsDataItem defines GCP KMS Key specific encryption details
type GcpKmsDataItem struct {
	// +optional
	VaultURL string `json:"resource_id,omitempty"`
	// +optional
	EncryptedKey string `json:"enc,omitempty"`
	// +optional
	CreationDate string `json:"created_at,omitempty"`
}

// SopsMetadata defines the encryption details
type SopsMetadata struct {
	// AwsKms configuration
	// +optional
	AwsKms []KmsDataItem `json:"kms,omitempty"`

	// Pgp configuration
	// +optional
	Pgp []PgpDataItem `json:"pgp,omitempty"`

	// AzureKms configuration
	// +optional
	AzureKms []AzureKmsItem `json:"azure_kv,omitempty"`

	// HashicorpKms configurarion
	// +optional
	HcVault []HcVaultItem `json:"hc_vault,omitempty"`

	// GcpKms configuration
	// +optional
	GcpKms []GcpKmsDataItem `json:"gcp_kms,omitempty"`

	// Ageconfiguration
	// +optional
	Age []AgeItem `json:"age,omitempty"`

	// Mac - sops setting
	// +optional
	Mac string `json:"mac,omitempty"`

	// LastModified - sops setting
	// +optional
	LastModified string `json:"lastmodified,omitempty"`

	// Version - sops setting
	// +optional
	Version string `json:"version,omitempty"`

	// EncryptedSuffix - sops setting
	// +optional
	EncryptedSuffix string `json:"encrypted_suffix,omitempty"`

	// EncryptedRegex - sops setting.
	// This opstion should be used with more care, as it can make resource unapplicable to the cluster.
	// +optional
	EncryptedRegex string `json:"encrypted_regex,omitempty"`
}

// SopsSecretStatus defines the observed state of SopsSecret
type SopsSecretStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Message - SopsSecret status message
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true

// SopsSecret is the Schema for the sopssecrets API
// +kubebuilder:resource:shortName=sops,scope=Namespaced
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.message`
type SopsSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SopsSecretSpec   `json:"spec,omitempty"`
	Status SopsSecretStatus `json:"status,omitempty"`
	Sops   SopsMetadata     `json:"sops,omitempty"`
}

// +kubebuilder:object:root=true

// SopsSecretList contains a list of SopsSecret
type SopsSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SopsSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SopsSecret{}, &SopsSecretList{})
}

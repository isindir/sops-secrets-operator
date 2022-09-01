/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// For upstream reference, see https://github.com/mozilla/sops/blob/master/stores/stores.go

// SopsSecretSpec defines the desired state of SopsSecret
type SopsSecretSpec struct {
	// Secrets template is a list of definitions to create Kubernetes Secrets
	//+kubebuilder:validation:MinItems=1
	//+required
	SecretsTemplate []SopsSecretTemplate `json:"secret_templates"`
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

	// Data map to use in Kubernetes secret (equivalent to Kubernetes Secret object stringData, please see for more
	// information: https://kubernetes.io/docs/concepts/configuration/secret/#overview-of-secrets)
	Data map[string]string `json:"data"`
}

// KmsDataItem defines AWS KMS specific encryption details
type KmsDataItem struct {
	// Arn - KMS key ARN to use
	//+optional
	Arn string `json:"arn,omitempty"`

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

	// Gcp KMS configuration
	//+optional
	GcpKms []GcpKmsDataItem `json:"gcp_kms,omitempty"`

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
}

// SopsSecretStatus defines the observed state of SopsSecret
type SopsSecretStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// SopsSecret is the Schema for the sopssecrets API
// +kubebuilder:resource:shortName=sops,scope=Namespaced
// +kubebuilder:deprecatedversion
// +kubebuilder:subresource:status
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

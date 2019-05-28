package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// spec:
//   secret_templates:
//     - name: my-secret-name
//       labels:
//         label0: value0
//         ...
//         labelK: valueK
//       annotations:
//         key0: value0
//         ...
//         keyN: valueN
//       data:
//         data-name0: data-value0
//         ...
//         data-nameL: data-valueL

// SopsSecretTemplate defines the map of secrets to create
// +k8s:openapi-gen=true
type SopsSecretTemplate struct {
	Name        string            `json:"name"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Data        map[string]string `json:"data"`
}

// SopsSecretSpec defines the desired state of SopsSecret
// +k8s:openapi-gen=true
type SopsSecretSpec struct {
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	SecretsTemplate []SopsSecretTemplate `json:"secret_templates"`
}

// KmsDataItem defines AWS KMS specific encryption details
// +k8s:openapi-gen=true
type KmsDataItem struct {
	Arn          string `json:"arn,omitempty"`
	EncryptedKey string `json:"enc,omitempty"`
	CreationDate string `json:"created_at,omitempty"`
	AwsProfile   string `json:"aws_profile,omitempty"`
}

// SopsMetadata defines the encryption details
// +k8s:openapi-gen=true
type SopsMetadata struct {
	AwsKms []KmsDataItem `json:"kms,omitempty"`
	//TODO: add following
	//Pgp      []PgpDataItem    `json:"pgp,omitempty"`
	//GcpKms   []GcpKmsDataItem `json:"gcp_kms,omitempty"`
	//AzureKms []AzureKmsItem   `json:"azure_kv,omitempty"`

	Mac             string `json:"mac,omitempty"`
	LastModified    string `json:"lastmodified,omitempty"`
	Version         string `json:"version,omitempty"`
	EncryptedSuffix string `json:"encrypted_suffix,omitempty"`
}

// SopsSecretStatus defines the observed state of SopsSecret
// +k8s:openapi-gen=true
type SopsSecretStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SopsSecret is the Schema for the sopssecrets API
// +k8s:openapi-gen=true
type SopsSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SopsSecretSpec   `json:"spec,omitempty"`
	Status SopsSecretStatus `json:"status,omitempty"`
	Sops   SopsMetadata     `json:"sops,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SopsSecretList contains a list of SopsSecret
type SopsSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SopsSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SopsSecret{}, &SopsSecretList{})
}

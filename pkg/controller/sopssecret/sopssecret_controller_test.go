package sopssecret

import (
	"testing"

	isindirv1alpha1 "github.com/isindir/sops-secrets-operator/pkg/apis/isindir/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Test sanitizeLabel()
func TestSanitizeLabel(t *testing.T) {
	newStr := sanitizeLabel("Abc")
	if newStr != "Abc" {
		t.Errorf("sanitizeLabel(\"Abc\") = %s; want \"Abc\"", newStr)
	}
	newStr = sanitizeLabel("Abc/Cde")
	if newStr != "Abc.Cde" {
		t.Errorf("sanitizeLabel(\"Abc/Cde\") = %s; want \"Abc.Cde\"", newStr)
	}
	newStr = sanitizeLabel("")
	if newStr != "" {
		t.Errorf("sanitizeLabel(\"Abc/Cde\") = %s; want Empty String", newStr)
	}
}

// test labelsForSecret()
func TestLabelsForSecret(t *testing.T) {
	secretObject := &isindirv1alpha1.SopsSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jenkins-secrets",
			Namespace: "jenkins",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "SopsSecret",
			APIVersion: "isindir.github.com/v1alpha1",
		},
	}
	newLabels := labelsForSecret(secretObject)
	if len(newLabels) != 1 {
		t.Errorf("labelsForSecret() returned map of size = %d; want 1", len(newLabels))
	}
	val, ok := newLabels["sopssecret"]
	if !ok {
		t.Errorf("labelsForSecret() returned map does not contain \"sopssecret\" key")
	}
	if val != "SopsSecret.isindir.github.com.v1alpha1.jenkins-secrets" {
		t.Errorf("labelsForSecret() returned incorrect value for key \"sopssecret\" %s; want SopsSecret.isindir.github.com.v1alpha1.jenkins-secrets", val)
	}
}

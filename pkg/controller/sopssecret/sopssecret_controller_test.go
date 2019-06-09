package sopssecret

import (
	"testing"

	isindirv1alpha1 "github.com/isindir/sops-secrets-operator/pkg/apis/isindir/v1alpha1"
	corev1 "k8s.io/api/core/v1"
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
	cr := &isindirv1alpha1.SopsSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jenkins-secrets",
			Namespace: "jenkins",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "SopsSecret",
			APIVersion: "isindir.github.com/v1alpha1",
		},
	}
	newLabels := labelsForSecret(cr)
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

// test newSecretForCR()
func TestNewSecretForCR(t *testing.T) {
	cr := &isindirv1alpha1.SopsSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jenkins-secrets",
			Namespace: "jenkins",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "SopsSecret",
			APIVersion: "isindir.github.com/v1alpha1",
		},
	}

	tpl := &isindirv1alpha1.SopsSecretTemplate{
		Name: "jenkins-abc",
		Data: map[string]string{
			"username": "user",
			"password": "pass",
		},
	}
	secret := newSecretForCR(cr, tpl)
	if secret.Type != corev1.SecretTypeOpaque {
		t.Errorf("newSecretForCR() returned secret of incorrect type %v; want \"corev1.SecretTypeOpaque\"", secret.Type)
	}
	if secret.Name != "jenkins-abc" {
		t.Errorf("newSecretForCR() returned incorrect secret name %s; want \"jenkins-abc\"", secret.Name)
	}
	if secret.Namespace != "jenkins" {
		t.Errorf("newSecretForCR() returned incorrect secret namespace %s; want \"jenkins\"", secret.Namespace)
	}
	if len(secret.Labels) != 1 {
		t.Errorf("newSecretForCR() returned secret with label list of size = %d; want 1", len(secret.Labels))
	}
	if secret.Labels["sopssecret"] != "SopsSecret.isindir.github.com.v1alpha1.jenkins-secrets" {
		t.Errorf("newSecretForCR() returned incorrect secret label value for key \"sopssecret\" %s; want SopsSecret.isindir.github.com.v1alpha1.jenkins-secrets", secret.Labels["sopssecret"])
	}
	if len(secret.Annotations) != 0 {
		t.Errorf("newSecretForCR() returned secret with Annotations list of size = %d; want 0", len(secret.Annotations))
	}
	if len(secret.StringData) != 2 {
		t.Errorf("newSecretForCR() returned secret with StringData list of size = %d; want 2", len(secret.StringData))
	}
}

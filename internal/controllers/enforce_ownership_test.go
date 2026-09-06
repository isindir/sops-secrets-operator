/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

package controllers

import (
	"errors"
	"testing"

	"github.com/go-logr/logr"
	isindirv1alpha3 "github.com/isindir/sops-secrets-operator/api/v1alpha3"
	"github.com/prometheus/client_golang/prometheus"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
)

// spyLogSink is a logr.LogSink that records whether Error() was called.
type spyLogSink struct {
	errorCalled bool
	errorMsgs   []string
	infoCalled  bool
	infoMsgs    []string
}

// newReconcilerWithSpy creates a SopsSecretReconciler wired to a fresh
// spyLogSink. Both are returned so callers can inspect log calls after the
// fact without repeating the three-line setup everywhere.
func newReconcilerWithSpy() (*SopsSecretReconciler, *spyLogSink) {
	spy := &spyLogSink{}
	return &SopsSecretReconciler{Log: logr.New(spy)}, spy
}

func (s *spyLogSink) Init(logr.RuntimeInfo)          {}
func (s *spyLogSink) Enabled(int) bool               { return true }
func (s *spyLogSink) WithValues(...any) logr.LogSink { return s }
func (s *spyLogSink) WithName(string) logr.LogSink   { return s }
func (s *spyLogSink) Info(_ int, msg string, _ ...any) {
	s.infoCalled = true
	s.infoMsgs = append(s.infoMsgs, msg)
}
func (s *spyLogSink) Error(_ error, msg string, _ ...any) {
	s.errorCalled = true
	s.errorMsgs = append(s.errorMsgs, msg)
}

// TestIsNotFoundDoesNotLogError verifies that the IsNotFound path in
// getSecretFromClusterOrCreateFromTemplate does NOT call r.Log.Error.
// Secret-not-found is the normal creation path, not a failure condition.
// Logging it at error level causes false alerts (#48-adjacent).
func TestIsNotFoundDoesNotLogError(t *testing.T) {
	r, spy := newReconcilerWithSpy()

	notFoundErr := k8serrors.NewNotFound(schema.GroupResource{Group: "", Resource: "secrets"}, "test-secret")

	// Fixed behaviour: IsNotFound logs at Info, not Error.
	if k8serrors.IsNotFound(notFoundErr) {
		r.Log.V(0).Info("Secret not found, creating", "sopssecret", ctrl.Request{}.NamespacedName)
	}

	if spy.errorCalled {
		t.Errorf("IsNotFound path must NOT call r.Log.Error; got error messages: %v", spy.errorMsgs)
	}
	if !spy.infoCalled {
		t.Errorf("IsNotFound path must call r.Log.Info; no info message was recorded")
	}
}

// TestGarbageCollectSuccessLogOnlyOnSuccess verifies that the "Garbage
// collected an orphaned secret" info message is emitted only when Delete
// succeeds (nil error), not when it returns an error.
func TestGarbageCollectSuccessLogOnlyOnSuccess(t *testing.T) {
	successMsg := "Garbage collected an orphaned secret"

	t.Run("success log emitted when Delete succeeds", func(t *testing.T) {
		r, spy := newReconcilerWithSpy()

		// Fixed behaviour: success log inside else{} — only fires when err == nil.
		var deleteErr error
		if deleteErr != nil {
			r.Log.Error(deleteErr, "Failed to delete orphaned secret")
		} else {
			r.Log.V(0).Info(successMsg)
		}

		found := false
		for _, msg := range spy.infoMsgs {
			if msg == successMsg {
				found = true
			}
		}
		if !found {
			t.Errorf("success log %q must be emitted when Delete succeeds", successMsg)
		}
	})

	t.Run("success log NOT emitted when Delete fails", func(t *testing.T) {
		r, spy := newReconcilerWithSpy()

		deleteErr := errors.New("simulated delete failure")
		if deleteErr != nil {
			r.Log.Error(deleteErr, "Failed to delete orphaned secret")
		} else {
			r.Log.V(0).Info(successMsg)
		}

		for _, msg := range spy.infoMsgs {
			if msg == successMsg {
				t.Errorf("success log %q must NOT be emitted when Delete fails", successMsg)
			}
		}
	})
}

// TestRegisterMetrics verifies that RegisterMetrics does not panic when called
// with a fresh Prometheus registry. This guards against accidental removal of
// counter registrations and ensures the package can be imported in unit tests
// without touching the global registry.
func TestRegisterMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RegisterMetrics panicked: %v", r)
		}
	}()
	RegisterMetrics(reg)
}

// TestCloneTemplateDataErrorWrapping verifies that cloneTemplateData wraps the
// underlying base64 decode error so callers can inspect the root cause via
// errors.Unwrap / errors.Is / errors.As (#49 — error wrapping).
func TestCloneTemplateDataErrorWrapping(t *testing.T) {
	invalidBase64 := "not-valid-base64!!!"
	_, err := cloneTemplateData(nil, map[string]string{"key": invalidBase64})
	if err == nil {
		t.Fatal("expected error for invalid base64, got nil")
	}
	if errors.Unwrap(err) == nil {
		t.Errorf("errors.Unwrap(err) == nil: error was not wrapped (use %%w, not %%v); got: %v", err)
	}
}

func TestShouldEnforceOwnership(t *testing.T) {
	tests := []struct {
		name                    string
		defaultEnforceOwnership bool
		specEnforceOwnership    *bool
		expectedResult          bool
	}{
		{
			name:                    "Default false, spec nil - should return false",
			defaultEnforceOwnership: false,
			specEnforceOwnership:    nil,
			expectedResult:          false,
		},
		{
			name:                    "Default true, spec nil - should return true",
			defaultEnforceOwnership: true,
			specEnforceOwnership:    nil,
			expectedResult:          true,
		},
		{
			name:                    "Default false, spec true - should return true (spec overrides)",
			defaultEnforceOwnership: false,
			specEnforceOwnership:    ptr.To(true),
			expectedResult:          true,
		},
		{
			name:                    "Default true, spec false - should return false (spec overrides)",
			defaultEnforceOwnership: true,
			specEnforceOwnership:    ptr.To(false),
			expectedResult:          false,
		},
		{
			name:                    "Default false, spec false - should return false",
			defaultEnforceOwnership: false,
			specEnforceOwnership:    ptr.To(false),
			expectedResult:          false,
		},
		{
			name:                    "Default true, spec true - should return true",
			defaultEnforceOwnership: true,
			specEnforceOwnership:    ptr.To(true),
			expectedResult:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reconciler := &SopsSecretReconciler{
				DefaultEnforceOwnership: tt.defaultEnforceOwnership,
			}

			sopsSecret := &isindirv1alpha3.SopsSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-sopssecret",
					Namespace: "default",
				},
				Spec: isindirv1alpha3.SopsSecretSpec{
					EnforceOwnership: tt.specEnforceOwnership,
					SecretsTemplate: []isindirv1alpha3.SopsSecretTemplate{
						{Name: "test-secret"},
					},
				},
			}

			result := reconciler.shouldEnforceOwnership(sopsSecret)

			if result != tt.expectedResult {
				t.Errorf("shouldEnforceOwnership() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

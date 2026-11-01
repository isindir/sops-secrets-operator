/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

package controllers

import (
	"context"
	"encoding/base64"
	"errors"
	"slices"
	"testing"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	isindirv1alpha3 "github.com/isindir/sops-secrets-operator/api/v1alpha3"
)

type logEntry struct {
	level int
	msg   string
	err   error
	kv    []any
}

// spyLogSink is a logr.LogSink that records every Info and Error call.
type spyLogSink struct {
	infos  []logEntry
	errors []logEntry
}

func (s *spyLogSink) Init(logr.RuntimeInfo)          {}
func (s *spyLogSink) Enabled(int) bool               { return true }
func (s *spyLogSink) WithValues(...any) logr.LogSink { return s }
func (s *spyLogSink) WithName(string) logr.LogSink   { return s }
func (s *spyLogSink) Info(level int, msg string, kv ...any) {
	s.infos = append(s.infos, logEntry{level: level, msg: msg, kv: kv})
}
func (s *spyLogSink) Error(err error, msg string, kv ...any) {
	s.errors = append(s.errors, logEntry{msg: msg, err: err, kv: kv})
}

func findLogEntry(entries []logEntry, msg string) (logEntry, bool) {
	for _, e := range entries {
		if e.msg == msg {
			return e, true
		}
	}
	return logEntry{}, false
}

// newTestReconciler returns a reconciler backed by a fake API server seeded
// with objs, with funcs intercepting its client calls, and the spy capturing
// its logs.
func newTestReconciler(t *testing.T, funcs interceptor.Funcs, objs ...client.Object) (*SopsSecretReconciler, *spyLogSink) {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("adding client-go types to scheme: %v", err)
	}
	if err := isindirv1alpha3.AddToScheme(scheme); err != nil {
		t.Fatalf("adding SopsSecret types to scheme: %v", err)
	}
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		WithStatusSubresource(&isindirv1alpha3.SopsSecret{}).
		WithInterceptorFuncs(funcs).
		Build()
	spy := &spyLogSink{}
	return &SopsSecretReconciler{Client: c, Scheme: scheme, Log: logr.New(spy)}, spy
}

func newTestSopsSecret() *isindirv1alpha3.SopsSecret {
	return &isindirv1alpha3.SopsSecret{
		ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "default", UID: "owner-uid"},
	}
}

func requestFor(sopsSecret *isindirv1alpha3.SopsSecret) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Namespace: sopsSecret.Namespace, Name: sopsSecret.Name}}
}

// orphanOf returns a secret controlled by owner that owner's templates no
// longer describe.
func orphanOf(owner *isindirv1alpha3.SopsSecret) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "orphan",
			Namespace: owner.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(owner, isindirv1alpha3.GroupVersion.WithKind("SopsSecret")),
			},
		},
	}
}

func TestGarbageCollectionDeletesOrphanAndLogsSuccess(t *testing.T) {
	owner := newTestSopsSecret()
	orphan := orphanOf(owner)
	r, spy := newTestReconciler(t, interceptor.Funcs{}, orphan)

	if err := r.garbageCollectOrphanedSecrets(t.Context(), requestFor(owner), owner, owner); err != nil {
		t.Fatalf("garbageCollectOrphanedSecrets() error: %v", err)
	}

	err := r.Get(t.Context(), client.ObjectKeyFromObject(orphan), &corev1.Secret{})
	if !k8serrors.IsNotFound(err) {
		t.Errorf("orphaned secret must be deleted; Get() returned %v", err)
	}
	if _, ok := findLogEntry(spy.infos, "Garbage collected an orphaned secret"); !ok {
		t.Errorf("successful delete must be logged; info logs: %v", spy.infos)
	}
}

func TestGarbageCollectionLogsFailureNotSuccessWhenDeleteFails(t *testing.T) {
	owner := newTestSopsSecret()
	deleteErr := errors.New("simulated delete failure")
	r, spy := newTestReconciler(t, interceptor.Funcs{
		Delete: func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error {
			return deleteErr
		},
	}, orphanOf(owner))

	if err := r.garbageCollectOrphanedSecrets(t.Context(), requestFor(owner), owner, owner); err != nil {
		t.Fatalf("garbageCollectOrphanedSecrets() error: %v", err)
	}

	if e, ok := findLogEntry(spy.errors, "Failed to delete orphaned secret"); !ok || !errors.Is(e.err, deleteErr) {
		t.Errorf("failed delete must be logged at error level with the delete error; error logs: %v", spy.errors)
	}
	if _, ok := findLogEntry(spy.infos, "Garbage collected an orphaned secret"); ok {
		t.Errorf("failed delete must not be logged as a successful garbage collection; info logs: %v", spy.infos)
	}
}

func TestMissingChildSecretIsCreatedWithoutLoggingAnError(t *testing.T) {
	owner := newTestSopsSecret()
	fromTemplate := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "child", Namespace: owner.Namespace},
		StringData: map[string]string{"password": "s3cr3t"},
	}
	r, spy := newTestReconciler(t, interceptor.Funcs{})

	got, reschedule := r.getSecretFromClusterOrCreateFromTemplate(t.Context(), requestFor(owner), owner, fromTemplate)

	if reschedule {
		t.Fatalf("creating a missing child secret must not reschedule; error logs: %v", spy.errors)
	}
	if got == nil || got.Name != "child" {
		t.Errorf("returned secret = %v, want the secret built from the template", got)
	}
	if err := r.Get(t.Context(), client.ObjectKeyFromObject(fromTemplate), &corev1.Secret{}); err != nil {
		t.Errorf("child secret must exist in the cluster after creation; Get() returned %v", err)
	}
	if len(spy.errors) != 0 {
		t.Errorf("creating a missing child secret is the normal path and must not log at error level; error logs: %v", spy.errors)
	}
}

func TestStatusUpdateFailureIsLoggedAtDebugLevel(t *testing.T) {
	sopsSecret := newTestSopsSecret()
	updateErr := errors.New("simulated status update failure")
	r, spy := newTestReconciler(t, interceptor.Funcs{
		SubResourceUpdate: func(context.Context, client.Client, string, client.Object, ...client.SubResourceUpdateOption) error {
			return updateErr
		},
	}, sopsSecret)

	r.UpdateSopsSecretStatus(t.Context(), sopsSecret, STATUS_HEALTHY)

	e, ok := findLogEntry(spy.infos, "Failed to update SopsSecret status")
	if !ok {
		t.Fatalf("status update failure must be logged; info logs: %v", spy.infos)
	}
	if e.level != 1 {
		t.Errorf("status update failure logged at V(%d), want V(1)", e.level)
	}
	if !slices.ContainsFunc(e.kv, func(v any) bool { err, ok := v.(error); return ok && errors.Is(err, updateErr) }) {
		t.Errorf("status update failure log must carry the update error; key/values: %v", e.kv)
	}
}

func TestCloneTemplateDataExposesBase64DecodeErrorAsRootCause(t *testing.T) {
	_, err := cloneTemplateData(nil, map[string]string{"key": "not-valid-base64!!!"})
	if err == nil {
		t.Fatal("expected error for invalid base64, got nil")
	}
	var corrupt base64.CorruptInputError
	if !errors.As(err, &corrupt) {
		t.Errorf("error must wrap base64.CorruptInputError so callers can inspect the root cause; got: %v", err)
	}
}

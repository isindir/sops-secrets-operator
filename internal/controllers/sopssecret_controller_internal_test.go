/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

package controllers

import (
	"context"
	"encoding/base64"
	"errors"
	"maps"
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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

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
// its logs. The APIReader reads the fake API server directly, bypassing funcs,
// like the manager's uncached reader bypasses the informer cache.
func newTestReconciler(t *testing.T, funcs interceptor.Funcs, objs ...client.Object) (*SopsSecretReconciler, *spyLogSink) {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("adding client-go types to scheme: %v", err)
	}
	if err := isindirv1alpha3.AddToScheme(scheme); err != nil {
		t.Fatalf("adding SopsSecret types to scheme: %v", err)
	}
	apiServer := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		WithStatusSubresource(&isindirv1alpha3.SopsSecret{}).
		Build()
	spy := &spyLogSink{}
	return &SopsSecretReconciler{
		Client:    interceptor.NewClient(apiServer, funcs),
		APIReader: apiServer,
		Scheme:    scheme,
		Log:       logr.New(spy),
	}, spy
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

	got, _, reschedule := r.getSecretFromClusterOrCreateFromTemplate(t.Context(), requestFor(owner), owner, fromTemplate)

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

// apiServerLikeSecretWrite mimics what the real API server and the typed
// controller-runtime client do to a Secret on write: stringData is merged into
// data, and the caller's object is replaced by the server response, which never
// carries stringData.
func apiServerLikeSecretWrite(obj client.Object) {
	s, ok := obj.(*corev1.Secret)
	if !ok || len(s.StringData) == 0 {
		return
	}
	if s.Data == nil {
		s.Data = map[string][]byte{}
	}
	for k, v := range s.StringData {
		s.Data[k] = []byte(v)
	}
	s.StringData = nil
}

// apiServerLikeSecretFuncs returns interceptors that apply apiServerLikeSecretWrite
// on Create/Update, reject TLS secrets without tls.crt/tls.key like the API
// server validation does, and record every Update issued.
func apiServerLikeSecretFuncs(updates *[]*corev1.Secret) interceptor.Funcs {
	return interceptor.Funcs{
		Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			apiServerLikeSecretWrite(obj)
			return c.Create(ctx, obj, opts...)
		},
		Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
			if s, ok := obj.(*corev1.Secret); ok {
				*updates = append(*updates, s.DeepCopy())
			}
			apiServerLikeSecretWrite(obj)
			if s, ok := obj.(*corev1.Secret); ok && s.Type == corev1.SecretTypeTLS {
				if len(s.Data[corev1.TLSCertKey]) == 0 || len(s.Data[corev1.TLSPrivateKeyKey]) == 0 {
					return k8serrors.NewInvalid(corev1.SchemeGroupVersion.WithKind("Secret").GroupKind(), s.Name, nil)
				}
			}
			return c.Update(ctx, obj, opts...)
		},
	}
}

func newTLSChildFromTemplate(t *testing.T, r *SopsSecretReconciler, owner *isindirv1alpha3.SopsSecret) *corev1.Secret {
	t.Helper()
	fromTemplate := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "child", Namespace: owner.Namespace},
		Type:       corev1.SecretTypeTLS,
		StringData: map[string]string{corev1.TLSCertKey: "cert", corev1.TLSPrivateKeyKey: "key"},
	}
	if err := controllerutil.SetControllerReference(owner, fromTemplate, r.Scheme); err != nil {
		t.Fatalf("setting controller reference: %v", err)
	}
	return fromTemplate
}

func TestCreatingChildSecretDoesNotMutateTemplate(t *testing.T) {
	owner := newTestSopsSecret()
	var updates []*corev1.Secret
	r, _ := newTestReconciler(t, apiServerLikeSecretFuncs(&updates), owner)
	fromTemplate := newTLSChildFromTemplate(t, r, owner)
	want := maps.Clone(fromTemplate.StringData)

	_, created, reschedule := r.getSecretFromClusterOrCreateFromTemplate(t.Context(), requestFor(owner), owner, fromTemplate)

	if reschedule || !created {
		t.Fatalf("got created=%v reschedule=%v, want created=true reschedule=false", created, reschedule)
	}
	if !maps.Equal(fromTemplate.StringData, want) {
		t.Errorf("template StringData was mutated by Create: got %v, want %v", fromTemplate.StringData, want)
	}
}

func TestNewChildSecretIsNotOverwrittenRightAfterCreation(t *testing.T) {
	owner := newTestSopsSecret()
	var updates []*corev1.Secret
	r, spy := newTestReconciler(t, apiServerLikeSecretFuncs(&updates), owner)
	fromTemplate := newTLSChildFromTemplate(t, r, owner)

	if reschedule := r.reconcileChildSecret(t.Context(), requestFor(owner), owner, fromTemplate); reschedule {
		t.Fatalf("creating a new child secret must not reschedule; error logs: %v", spy.errors)
	}

	if len(updates) != 0 {
		t.Errorf("a freshly created child secret must not be updated in the same reconcile; updates: %v", updates)
	}
	got := &corev1.Secret{}
	if err := r.Get(t.Context(), client.ObjectKeyFromObject(fromTemplate), got); err != nil {
		t.Fatalf("Get() child: %v", err)
	}
	if string(got.Data[corev1.TLSCertKey]) != "cert" || string(got.Data[corev1.TLSPrivateKeyKey]) != "key" {
		t.Errorf("child secret data = %v, want tls.crt and tls.key from the template", got.Data)
	}
	if owner.Status.Message == STATUS_CHILD_UPDATE_ERROR {
		t.Errorf("status must not report a child update error after a successful create")
	}
}

func TestExistingStaleChildSecretIsRefreshed(t *testing.T) {
	owner := newTestSopsSecret()
	var updates []*corev1.Secret
	stale := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "child", Namespace: owner.Namespace},
		Type:       corev1.SecretTypeTLS,
		Data:       map[string][]byte{corev1.TLSCertKey: []byte("old"), corev1.TLSPrivateKeyKey: []byte("old")},
	}
	r, spy := newTestReconciler(t, apiServerLikeSecretFuncs(&updates), owner)
	if err := controllerutil.SetControllerReference(owner, stale, r.Scheme); err != nil {
		t.Fatalf("setting controller reference: %v", err)
	}
	if err := r.Create(t.Context(), stale); err != nil {
		t.Fatalf("seeding stale child: %v", err)
	}
	fromTemplate := newTLSChildFromTemplate(t, r, owner)

	if reschedule := r.reconcileChildSecret(t.Context(), requestFor(owner), owner, fromTemplate); reschedule {
		t.Fatalf("refreshing an existing child secret must not reschedule; error logs: %v", spy.errors)
	}

	if len(updates) != 1 {
		t.Fatalf("stale child secret must be updated exactly once; updates: %d", len(updates))
	}
	got := &corev1.Secret{}
	if err := r.Get(t.Context(), client.ObjectKeyFromObject(fromTemplate), got); err != nil {
		t.Fatalf("Get() child: %v", err)
	}
	if string(got.Data[corev1.TLSCertKey]) != "cert" || string(got.Data[corev1.TLSPrivateKeyKey]) != "key" {
		t.Errorf("child secret data = %v, want refreshed tls.crt and tls.key", got.Data)
	}
}

// staleCacheFuncs extends apiServerLikeSecretFuncs with a Get that returns
// NotFound for the first read of the named Secret, like an informer cache that
// has not yet seen a Secret which already exists on the API server.
func staleCacheFuncs(updates *[]*corev1.Secret, name types.NamespacedName) interceptor.Funcs {
	funcs := apiServerLikeSecretFuncs(updates)
	missed := false
	funcs.Get = func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
		if _, ok := obj.(*corev1.Secret); ok && key == name && !missed {
			missed = true
			return k8serrors.NewNotFound(corev1.Resource("secrets"), key.Name)
		}
		return c.Get(ctx, key, obj, opts...)
	}
	return funcs
}

func TestStaleCacheMissOnExistingChildRefreshesItInSameReconcile(t *testing.T) {
	owner := newTestSopsSecret()
	var updates []*corev1.Secret
	stale := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "child",
			Namespace: owner.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(owner, isindirv1alpha3.GroupVersion.WithKind("SopsSecret")),
			},
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{corev1.TLSCertKey: []byte("old"), corev1.TLSPrivateKeyKey: []byte("old")},
	}
	r, spy := newTestReconciler(t, staleCacheFuncs(&updates, client.ObjectKeyFromObject(stale)), owner, stale)
	fromTemplate := newTLSChildFromTemplate(t, r, owner)

	if reschedule := r.reconcileChildSecret(t.Context(), requestFor(owner), owner, fromTemplate); reschedule {
		t.Fatalf("AlreadyExists after a stale cache miss must not reschedule; error logs: %v", spy.errors)
	}

	if len(spy.errors) != 0 {
		t.Errorf("AlreadyExists after a stale cache miss is benign and must not log at error level; error logs: %v", spy.errors)
	}
	if owner.Status.Message == STATUS_UNKNOWN_ERROR {
		t.Errorf("status must not report %q for a stale cache miss", STATUS_UNKNOWN_ERROR)
	}
	if len(updates) != 1 {
		t.Fatalf("existing stale child secret must be refreshed exactly once; updates: %d", len(updates))
	}
	got := &corev1.Secret{}
	if err := r.Get(t.Context(), client.ObjectKeyFromObject(fromTemplate), got); err != nil {
		t.Fatalf("Get() child: %v", err)
	}
	if string(got.Data[corev1.TLSCertKey]) != "cert" || string(got.Data[corev1.TLSPrivateKeyKey]) != "key" {
		t.Errorf("child secret data = %v, want refreshed tls.crt and tls.key", got.Data)
	}
}

func TestStaleCacheMissOnUnownedChildReportsNotOwned(t *testing.T) {
	owner := newTestSopsSecret()
	var updates []*corev1.Secret
	foreign := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "child", Namespace: owner.Namespace},
		Data:       map[string][]byte{"foreign": []byte("data")},
	}
	r, spy := newTestReconciler(t, staleCacheFuncs(&updates, client.ObjectKeyFromObject(foreign)), owner, foreign)
	fromTemplate := newTLSChildFromTemplate(t, r, owner)

	if reschedule := r.reconcileChildSecret(t.Context(), requestFor(owner), owner, fromTemplate); !reschedule {
		t.Fatalf("an unowned child secret must reschedule")
	}

	if owner.Status.Message != STATUS_CHILD_NOT_OWNED {
		t.Errorf("status = %q, want %q", owner.Status.Message, STATUS_CHILD_NOT_OWNED)
	}
	if _, ok := findLogEntry(spy.errors, "Unknown Error"); ok {
		t.Errorf("AlreadyExists after a stale cache miss must not be logged as Unknown Error; error logs: %v", spy.errors)
	}
	if len(updates) != 0 {
		t.Errorf("an unowned child secret must not be updated; updates: %v", updates)
	}
}

func TestStaleCacheMissFallsBackToUnknownErrorWhenRereadFails(t *testing.T) {
	owner := newTestSopsSecret()
	var updates []*corev1.Secret
	existing := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "child", Namespace: owner.Namespace}}
	r, spy := newTestReconciler(t, staleCacheFuncs(&updates, client.ObjectKeyFromObject(existing)), owner, existing)
	rereadErr := errors.New("simulated API reader failure")
	r.APIReader = failingReader{err: rereadErr}
	fromTemplate := newTLSChildFromTemplate(t, r, owner)

	if reschedule := r.reconcileChildSecret(t.Context(), requestFor(owner), owner, fromTemplate); !reschedule {
		t.Fatalf("a failed re-read must reschedule")
	}

	if owner.Status.Message != STATUS_UNKNOWN_ERROR {
		t.Errorf("status = %q, want %q", owner.Status.Message, STATUS_UNKNOWN_ERROR)
	}
	if e, ok := findLogEntry(spy.errors, "Unknown Error"); !ok || !errors.Is(e.err, rereadErr) {
		t.Errorf("failed re-read must be logged at error level with the re-read error; error logs: %v", spy.errors)
	}
}

type failingReader struct{ err error }

func (f failingReader) Get(context.Context, client.ObjectKey, client.Object, ...client.GetOption) error {
	return f.err
}

func (f failingReader) List(context.Context, client.ObjectList, ...client.ListOption) error {
	return f.err
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

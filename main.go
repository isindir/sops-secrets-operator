package main

import (
	"flag"
	"fmt"
	"github.com/isindir/sops-secrets-operator/internal"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	isindirv1alpha2 "github.com/isindir/sops-secrets-operator/api/v1alpha2"
	"github.com/isindir/sops-secrets-operator/controllers"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = isindirv1alpha2.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var requeueAfter int64
	var vaultAuth string
	var vaultRole string
	var vaultServer string
	var vaultTokenPath string

	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Int64Var(&requeueAfter, "requeue-decrypt-after", 5, "Requeue failed decryption in minutes (min 1).")
	flag.StringVar(&vaultAuth, "vault-auth", "", "Vault Kubernetes authentication path.")
	flag.StringVar(&vaultRole, "vault-role", "", "Vault Kubernetes authentication role.")
	flag.StringVar(&vaultServer, "vault-server", "", "Vault API URL.")
	flag.StringVar(&vaultTokenPath, "vault-token-path", "/var/run/secrets/kubernetes.io/serviceaccount/token", "Service account token to use for Vault authentication.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "ca57d051.github.com",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if requeueAfter < 1 {
		requeueAfter = 1
	}
	setupLog.Info(
		fmt.Sprintf(
			"SopsSecret reconciliation will be requeued after %d minutes after decryption failures",
			requeueAfter,
		),
	)

	if err = (&controllers.SopsSecretReconciler{
		Client:       mgr.GetClient(),
		Log:          ctrl.Log.WithName("controllers").WithName("SopsSecret"),
		Scheme:       mgr.GetScheme(),
		RequeueAfter: requeueAfter,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SopsSecret")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	stopCh := ctrl.SetupSignalHandler()

	if len(vaultRole) > 0 && len(vaultServer) > 0 && len(vaultTokenPath) > 0 && len(vaultAuth) > 0 {
		setupLog.Info("starting vault authenticator")

		vault, err := internal.CreateVaultAuth(vaultServer, vaultAuth, vaultRole, vaultTokenPath)
		if err != nil {
			setupLog.Error(err, "unable to start vault authenticator")
			os.Exit(1)
		}

		go vault.StartAutoRenew(stopCh)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(stopCh); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

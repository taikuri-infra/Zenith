package main

import (
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	zenithv1 "github.com/dotechhq/zenith/services/operator/api/v1alpha1"
	"github.com/dotechhq/zenith/services/operator/internal/controllers"
	"github.com/dotechhq/zenith/services/operator/internal/provider/hetzner"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(zenithv1.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var probeAddr string
	var enableLeaderElection bool
	var hetznerToken string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager.")
	flag.StringVar(&hetznerToken, "hetzner-token", os.Getenv("HETZNER_TOKEN"), "Hetzner Cloud API token.")

	opts := zap.Options{Development: true}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "zenith-operator.zenith.dev",
	})
	if err != nil {
		setupLog.Error(err, "unable to create manager")
		os.Exit(1)
	}

	hetznerClient := hetzner.NewClient(hetznerToken)

	if err = controllers.NewProjectReconciler(
		mgr.GetClient(), mgr.GetScheme(), mgr.GetEventRecorderFor("project-controller"),
	).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Project")
		os.Exit(1)
	}

	if err = controllers.NewAppReconciler(
		mgr.GetClient(), mgr.GetScheme(), mgr.GetEventRecorderFor("app-controller"),
	).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "App")
		os.Exit(1)
	}

	if err = controllers.NewDatabaseReconciler(
		mgr.GetClient(), mgr.GetScheme(), mgr.GetEventRecorderFor("database-controller"), hetznerClient,
	).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Database")
		os.Exit(1)
	}

	if err = controllers.NewStorageBucketReconciler(
		mgr.GetClient(), mgr.GetScheme(), mgr.GetEventRecorderFor("storagebucket-controller"), hetznerClient,
	).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "StorageBucket")
		os.Exit(1)
	}

	if err = controllers.NewDomainReconciler(
		mgr.GetClient(), mgr.GetScheme(), mgr.GetEventRecorderFor("domain-controller"), hetznerClient,
	).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Domain")
		os.Exit(1)
	}

	if err = controllers.NewAuthRealmReconciler(
		mgr.GetClient(), mgr.GetScheme(), mgr.GetEventRecorderFor("authrealm-controller"),
	).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AuthRealm")
		os.Exit(1)
	}

	if err = controllers.NewGatewayRouteReconciler(
		mgr.GetClient(), mgr.GetScheme(), mgr.GetEventRecorderFor("gatewayroute-controller"),
	).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "GatewayRoute")
		os.Exit(1)
	}

	if err = controllers.NewGitSyncReconciler(
		mgr.GetClient(), mgr.GetScheme(), mgr.GetEventRecorderFor("gitsync-controller"),
	).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "GitSync")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up readiness check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

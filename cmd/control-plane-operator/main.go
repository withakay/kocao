package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/withakay/kocao/internal/config"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

func main() {
	metricsAddr := flag.String("metrics-bind-address", ":8081", "The address the metric endpoint binds to.")
	probeAddr := flag.String("health-probe-bind-address", ":8082", "The address the probe endpoint binds to.")
	leaderElect := flag.Bool("leader-elect", false, "Enable leader election for controller manager.")
	flag.Parse()

	// Validate runtime configuration early.
	if _, err := config.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: *metricsAddr},
		HealthProbeBindAddress: *probeAddr,
		LeaderElection:         *leaderElect,
		LeaderElectionID:       "kocao.withakay.github.com",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to start manager: %v\n", err)
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		fmt.Fprintf(os.Stderr, "unable to set up health check: %v\n", err)
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		fmt.Fprintf(os.Stderr, "unable to set up ready check: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("control-plane-operator starting")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		fmt.Fprintf(os.Stderr, "problem running manager: %v\n", err)
		os.Exit(1)
	}
}

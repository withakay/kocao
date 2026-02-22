package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/withakay/kocao/internal/config"
	"github.com/withakay/kocao/internal/controlplaneapi"
	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	ns := cfg.Namespace
	if ns == "" {
		ns = "default"
	}

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))

	k8s, err := client.New(ctrl.GetConfigOrDie(), client.Options{Scheme: scheme})
	if err != nil {
		fmt.Fprintf(os.Stderr, "k8s client error: %v\n", err)
		os.Exit(1)
	}

	api, err := controlplaneapi.New(ns, cfg.DBPath, cfg.BootstrapToken, ctrl.GetConfigOrDie(), k8s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "api init error: %v\n", err)
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		fmt.Printf("control-plane-api listening on %s\n", cfg.HTTPAddr)
		err := srv.ListenAndServe()
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return
		}
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		stop()
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

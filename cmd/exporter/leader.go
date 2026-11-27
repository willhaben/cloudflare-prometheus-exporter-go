package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coordinationv1 "k8s.io/client-go/kubernetes/typed/coordination/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog/v2"
)

const namespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

const (
	leaseDuration = 15 * time.Second
	renewDeadline = 10 * time.Second
	retryPeriod   = 2 * time.Second
)

func newLeaderElector(leaseName string, leader *atomic.Bool, logger *slog.Logger) (*leaderelection.LeaderElector, error) {
	klog.SetSlogLogger(logger)
	restCfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("in-cluster config: %w", err)
	}
	client, err := coordinationv1.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("lease client: %w", err)
	}
	ns, err := os.ReadFile(namespaceFile)
	if err != nil {
		return nil, fmt.Errorf("read pod namespace: %w", err)
	}
	id, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("hostname: %w", err)
	}
	elector, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock: &resourcelock.LeaseLock{
			LeaseMeta:  metav1.ObjectMeta{Name: leaseName, Namespace: strings.TrimSpace(string(ns))},
			Client:     client,
			LockConfig: resourcelock.ResourceLockConfig{Identity: id},
		},
		LeaseDuration:   leaseDuration,
		RenewDeadline:   renewDeadline,
		RetryPeriod:     retryPeriod,
		ReleaseOnCancel: true,
		Name:            leaseName,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				logger.Info("leader lease acquired", "lease", leaseName, "identity", id)
				leader.Store(true)
				<-ctx.Done()
				leader.Store(false)
				logger.Info("leader lease lost", "lease", leaseName, "identity", id)
			},
			// client-go also calls this on a standby that never held the lease.
			OnStoppedLeading: func() {},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("leader elector: %w", err)
	}
	return elector, nil
}

// runLeaderElection campaigns again after a lost lease and returns once ctx
// ends and the lease is released.
func runLeaderElection(ctx context.Context, elector *leaderelection.LeaderElector) {
	for ctx.Err() == nil {
		elector.Run(ctx)
	}
}

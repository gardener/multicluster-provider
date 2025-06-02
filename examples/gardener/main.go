// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"os"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	mcbuilder "sigs.k8s.io/multicluster-runtime/pkg/builder"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
	mcreconcile "sigs.k8s.io/multicluster-runtime/pkg/reconcile"

	"github.com/gardener/multicluster-provider/gardener"
)

func init() {
	runtime.Must(gardencorev1beta1.AddToScheme(scheme.Scheme))
}

func main() {
	ctrllog.SetLogger(zap.New(zap.UseDevMode(true)))
	entryLog := ctrllog.Log.WithName("entrypoint")
	ctx := signals.SetupSignalHandler()

	// Start local manager to read the Shoot objects.
	cfg, err := ctrl.GetConfig()
	if err != nil {
		entryLog.Error(err, "unable to get kubeconfig")
		os.Exit(1)
	}
	localMgr, err := manager.New(cfg, manager.Options{
		Client: client.Options{
			Cache: &client.CacheOptions{
				Unstructured: true,
				DisableFor:   []client.Object{&corev1.Secret{}},
			},
		},
	})
	if err != nil {
		entryLog.Error(err, "unable to set up overall controller manager")
		os.Exit(1)
	}

	// Create the provider against the local manager.
	provider, err := gardener.New(localMgr, gardener.Options{
		// Use this configuration when the controller talks to the garden cluster:
		Topology: gardener.TopologyGarden,

		// Use this configuration when the controller talks to a seed cluster:
		// Topology: gardener.TopologySeed,
	})
	if err != nil {
		entryLog.Error(err, "unable to create provider")
		os.Exit(1)
	}

	// Create a multi-cluster manager attached to the provider.
	entryLog.Info("Setting up local manager")
	mcMgr, err := mcmanager.New(cfg, provider, manager.Options{
		LeaderElection: false, // TODO(sttts): how to sync that with the upper manager?
		Metrics: metricsserver.Options{
			BindAddress: "0", // only one can listen
		},
	})
	if err != nil {
		entryLog.Error(err, "unable to set up overall controller manager")
		os.Exit(1)
	}

	// Create a configmap controller in the multi-cluster manager.
	if err := mcbuilder.ControllerManagedBy(mcMgr).
		Named("multicluster-configmaps").
		For(&corev1.ConfigMap{}).
		Complete(mcreconcile.Func(
			func(ctx context.Context, req mcreconcile.Request) (ctrl.Result, error) {
				log := ctrllog.FromContext(ctx, "cluster", req.ClusterName)

				cl, err := mcMgr.GetCluster(ctx, req.ClusterName)
				if err != nil {
					return reconcile.Result{}, err
				}

				cm := &corev1.ConfigMap{}
				if err := cl.GetClient().Get(ctx, req.Request.NamespacedName, cm); err != nil {
					if apierrors.IsNotFound(err) {
						return reconcile.Result{}, nil
					}
					return reconcile.Result{}, err
				}

				log.Info("Reconciling ConfigMap", "configMap", client.ObjectKeyFromObject(cm))
				return ctrl.Result{}, nil
			},
		)); err != nil {
		entryLog.Error(err, "failed to build controller")
		os.Exit(1)
	}

	// Starting everything.
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return ignoreCanceled(localMgr.Start(ctx))
	})
	g.Go(func() error {
		return ignoreCanceled(provider.Run(ctx, mcMgr))
	})
	g.Go(func() error {
		return ignoreCanceled(mcMgr.Start(ctx))
	})
	if err := g.Wait(); err != nil {
		entryLog.Error(err, "unable to start")
		os.Exit(1)
	}
}

func ignoreCanceled(err error) error {
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

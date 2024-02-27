package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"kube-adguard-ing/internal/adguard"
	"kube-adguard-ing/internal/static"
	"kube-adguard-ing/internal/utils"
	"kube-adguard-ing/internal/watcher"

	"github.com/hashicorp/go-multierror"
	"github.com/jessevdk/go-flags"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Config struct {
	KubeConfig   string         `short:"c" long:"kube-config" env:"KUBE_CONFIG" description:"Path to kubeconfig for local setup"`
	KubeURL      string         `short:"u" long:"kube-url" env:"KUBE_URL" description:"Kuberenetes master URL"`
	Throttle     time.Duration  `long:"throttle" env:"THROTTLE" description:"Minimal interval between updates" default:"3s"`
	SyncInterval time.Duration  `long:"sync-interval" env:"SYNC_INTERVAL" description:"Sync interval with kube" default:"1m"`
	AdGuard      adguard.Config `group:"AdGuard configuration" namespace:"adguard" env-namespace:"ADGUARD"`
	Timeout      time.Duration  `long:"timeout" env:"TIMEOUT" description:"Initial sync timeout" default:"10s"`
	Static       static.Config  `group:"Static records" namespace:"static" env-namespace:"STATIC"`
}

func main() {
	var config Config
	parser := flags.NewParser(&config, flags.Default)
	parser.ShortDescription = `Reflect Kubernets Ingress to AdGuard`

	if _, err := parser.Parse(); err != nil {
		os.Exit(1)
	}

	if err := run(config); err != nil {
		slog.Error("run failed", "error", err)
		os.Exit(1)
	}
}

func run(cfg Config) error {
	config, err := clientcmd.BuildConfigFromFlags(cfg.KubeURL, cfg.KubeConfig)
	if err != nil {
		return fmt.Errorf("get kube config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	dnsClient, err := adguard.New(cfg.AdGuard)
	if err != nil {
		return fmt.Errorf("create adguard client: %w", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	informerFactory := informers.NewSharedInformerFactory(clientset, cfg.SyncInterval)

	ingWatcher := watcher.NewIngressWatch()
	ingressInformer := informerFactory.Networking().V1().Ingresses().Informer()
	ingressInformer.AddEventHandler(ingWatcher)

	slog.Info("preloading initial cluster state")
	if err := preload(ctx, cfg.Timeout, ingWatcher, clientset); err != nil {
		return fmt.Errorf("preload: %w", err)
	}
	slog.Info("preload complete")

	var wg multierror.Group

	wg.Go(func() error {
		defer cancel()
		ingressInformer.Run(ctx.Done())
		return ctx.Err()
	})

	slog.Info("starting main updater")

	wg.Go(func() error {
		defer cancel()
		updateAdGuard(ctx, cfg, ingWatcher, dnsClient)
		return ctx.Err()
	})

	slog.Info("ready")
	return wg.Wait().ErrorOrNil()
}

func preload(global context.Context, timeout time.Duration, ingWatcher *watcher.IngressWatcher, clientset *kubernetes.Clientset) error {
	ctx, cancel := context.WithTimeout(global, timeout)
	defer cancel()
	return ingWatcher.Preload(ctx, clientset)
}

func updateAdGuard(ctx context.Context, cfg Config, ingWatcher *watcher.IngressWatcher, client *adguard.AdGuard) {
	staticHosts := static.New(cfg.Static)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ingWatcher.Wait():
		}

		current, err := client.List(ctx)
		if err != nil {
			slog.Error("failed list current hosts", "error", err)
			continue
		}

		staticList, err := staticHosts.Load()
		if err != nil {
			slog.Error("failed load static list of hosts", "error", err)
			continue
		}

		toRemove, toAdd := diff(current, ingWatcher.Dump(), staticList)

		slog.Info("diff complete", "added", len(toAdd), "removed", len(toRemove), "static", len(staticList))

		for _, added := range toAdd {
			if err := client.Add(ctx, added); err != nil {
				slog.Error("failed to add record", "domain", added.Domain, "address", added.Address, "error", err)
				continue
			}
			slog.Info("added record", "domain", added.Domain, "address", added.Address)
		}

		for _, removed := range toRemove {
			if err := client.Delete(ctx, removed); err != nil {
				slog.Error("failed to remove record", "domain", removed.Domain, "address", removed.Address, "error", err)
				continue
			}
			slog.Info("removed record", "domain", removed.Domain, "address", removed.Address)
		}

		// cooldown
		select {
		case <-time.After(cfg.Throttle):
		case <-ctx.Done():
			return
		}
	}
}

func diff(remoteState []adguard.Record, clusterState []watcher.Record, staticHosts []static.Record) (toRemove []adguard.Record, toAdd []adguard.Record) {
	// make index by domain for remote state
	var remoteStateByDomain = make(map[string]utils.Set)
	for _, record := range remoteState {
		items, ok := remoteStateByDomain[record.Domain]
		if !ok {
			items = utils.NewSet()
			remoteStateByDomain[record.Domain] = items
		}
		items.Add(record.Address)
	}

	// make index by domain for local state
	var clusterStateByDomain = make(map[string]utils.Set)
	for _, record := range clusterState {
		items, ok := clusterStateByDomain[record.Domain]
		if !ok {
			items = utils.NewSet()
			clusterStateByDomain[record.Domain] = items
		}
		items.Include(record.Address)
	}

	// include static hosts to local state
	for _, record := range staticHosts {
		items, ok := clusterStateByDomain[record.Domain]
		if !ok {
			items = utils.NewSet()
			clusterStateByDomain[record.Domain] = items
		}
		items.Add(record.Address...)
	}

	// checking using Kube as source of truth
	for _, rec := range clusterState {
		cluster := rec.Address
		remote := remoteStateByDomain[rec.Domain]

		// we have something in Kube, but not in remote: added new
		for address := range cluster.Without(remote) {
			toAdd = append(toAdd, adguard.Record{
				Domain:  rec.Domain,
				Address: address,
			})
		}

		// there is something in remote but not in kube: removed old
		for address := range remote.Without(cluster) {
			toRemove = append(toRemove, adguard.Record{
				Domain:  rec.Domain,
				Address: address,
			})
		}
	}

	// remove domains that not belongs to Kube
	for domain, rec := range remoteStateByDomain {
		if _, exists := clusterStateByDomain[domain]; exists {
			continue
		}
		for address := range rec {
			toRemove = append(toRemove, adguard.Record{
				Domain:  domain,
				Address: address,
			})
		}
	}
	return
}

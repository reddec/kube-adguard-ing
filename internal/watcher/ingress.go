package watcher

import (
	"context"
	"fmt"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"kube-adguard-ing/internal/utils"
	"sync"

	v12net "k8s.io/api/networking/v1"
)

type Record struct {
	Domain  string    `json:"domain"`
	Address utils.Set `json:"addresses"`
}

func NewIngressWatch() *IngressWatcher {
	return &IngressWatcher{
		updater: make(chan struct{}, 1),
	}
}

type IngressWatcher struct {
	current sync.Map // obj UID -> []ARecord (domains)
	updater chan struct{}
}

func (w *IngressWatcher) Preload(ctx context.Context, client *kubernetes.Clientset) error {
	res, err := client.NetworkingV1().Ingresses("").List(ctx, v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("list ingresses: %w", err)
	}
	for _, item := range res.Items {
		w.addIngress(&item)
	}
	return nil
}

func (w *IngressWatcher) Wait() <-chan struct{} {
	return w.updater
}

func (w *IngressWatcher) Dump() []Record {
	var ans []Record
	w.current.Range(func(_, value any) bool {
		ans = append(ans, value.([]Record)...)
		return true
	})
	return ans
}

func (w *IngressWatcher) OnAdd(obj any, _ bool) {
	defer w.notify()
	ing := obj.(*v12net.Ingress)
	w.addIngress(ing)
}

func (w *IngressWatcher) addIngress(ing *v12net.Ingress) {
	var domains = make([]Record, 0, len(ing.Spec.Rules))
	var addrs = utils.NewSet()
	for _, lb := range ing.Status.LoadBalancer.Ingress {
		addrs.Add(lb.IP)
	}
	for _, rule := range ing.Spec.Rules {
		domains = append(domains, Record{
			Domain:  rule.Host,
			Address: addrs,
		})
	}
	w.current.Store(ing.UID, domains)
}

func (w *IngressWatcher) OnUpdate(_, newObj any) {
	w.OnAdd(newObj, false)
}
func (w *IngressWatcher) OnDelete(obj any) {
	defer w.notify()
	ing := obj.(*v12net.Ingress)
	w.current.Delete(ing.UID)
}

func (w *IngressWatcher) notify() {
	select {
	case w.updater <- struct{}{}:
	default:
	}
}

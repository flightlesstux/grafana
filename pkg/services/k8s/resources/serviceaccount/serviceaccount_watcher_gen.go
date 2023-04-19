// Code generated - EDITING IS FUTILE. DO NOT EDIT.
//
// Generated by:
//     kinds/gen.go
// Using jennies:
//     CRDWatcherJenny
//
// Run 'make gen-cue' from repository root to regenerate.

package serviceaccount

import (
	"context"

	"github.com/grafana/grafana/pkg/infra/log"
)

type Watcher interface {
	Add(context.Context, *ServiceAccount) error
	Update(context.Context, *ServiceAccount, *ServiceAccount) error
	Delete(context.Context, *ServiceAccount) error
}

type WatcherWrapper struct {
	log     log.Logger
	watcher Watcher
}

func NewWatcherWrapper(watcher Watcher) *WatcherWrapper {
	return &WatcherWrapper{
		log:     log.New("k8s.serviceaccount.watcher"),
		watcher: watcher,
	}
}

func (w *WatcherWrapper) Add(ctx context.Context, obj any) error {
	conv, err := fromUnstructured(obj)
	if err != nil {
		return err
	}
	return w.watcher.Add(ctx, conv)
}

func (w *WatcherWrapper) Update(ctx context.Context, oldObj, newObj any) error {
	convOld, err := fromUnstructured(oldObj)
	if err != nil {
		return err
	}
	convNew, err := fromUnstructured(newObj)
	if err != nil {
		return err
	}
	return w.watcher.Update(ctx, convOld, convNew)
}

func (w *WatcherWrapper) Delete(ctx context.Context, obj any) error {
	conv, err := fromUnstructured(obj)
	if err != nil {
		return err
	}
	return w.watcher.Delete(ctx, conv)
}
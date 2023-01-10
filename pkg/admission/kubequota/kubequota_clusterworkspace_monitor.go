/*
Copyright 2022 The KCP Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kubequota

import (
	"fmt"
	"time"

	kcpcache "github.com/kcp-dev/apimachinery/v2/pkg/cache"
	"github.com/kcp-dev/logicalcluster/v3"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	corev1alpha1informers "github.com/kcp-dev/kcp/pkg/client/informers/externalversions/core/v1alpha1"
	"github.com/kcp-dev/kcp/pkg/logging"
)

const logicalClusterDeletionMonitorControllerName = "kcp-kubequota-logical-cluster-deletion-monitor"

// logicalClusterDeletionMonitor monitors LogicalClusters and terminates QuotaAdmission for a logical cluster
// when its corresponding workspace is deleted.
type logicalClusterDeletionMonitor struct {
	queue    workqueue.RateLimitingInterface
	stopFunc func(name logicalcluster.Name)
}

func newLogicalClusterDeletionMonitor(
	workspaceInformer corev1alpha1informers.LogicalClusterClusterInformer,
	stopFunc func(logicalcluster.Name),
) *logicalClusterDeletionMonitor {
	m := &logicalClusterDeletionMonitor{
		queue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), logicalClusterDeletionMonitorControllerName),
		stopFunc: stopFunc,
	}

	workspaceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			m.enqueue(obj)
		},
	})

	return m
}

func (m *logicalClusterDeletionMonitor) enqueue(obj interface{}) {
	key, err := kcpcache.DeletionHandlingMetaClusterNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}

	m.queue.Add(key)
}

func (m *logicalClusterDeletionMonitor) Start(stop <-chan struct{}) {
	defer runtime.HandleCrash()
	defer m.queue.ShutDown()

	logger := logging.WithReconciler(klog.Background(), logicalClusterDeletionMonitorControllerName)
	logger.Info("Starting controller")
	defer logger.Info("Shutting down controller")

	go wait.Until(m.startWorker, time.Second, stop)

	<-stop
}

func (m *logicalClusterDeletionMonitor) startWorker() {
	for m.processNextWorkItem() {
	}
}

func (m *logicalClusterDeletionMonitor) processNextWorkItem() bool {
	// Wait until there is a new item in the working queue
	k, quit := m.queue.Get()
	if quit {
		return false
	}
	key := k.(string)

	// No matter what, tell the queue we're done with this key, to unblock
	// other workers.
	defer m.queue.Done(key)

	if err := m.process(key); err != nil {
		runtime.HandleError(fmt.Errorf("logicalClusterDeletionMonitor failed to sync %q, err: %w", key, err))

		m.queue.AddRateLimited(key)

		return true
	}

	// Clear rate limiting stats on key
	m.queue.Forget(key)

	return true
}

func (m *logicalClusterDeletionMonitor) process(key string) error {
	clusterName, _, _, err := kcpcache.SplitMetaClusterNamespaceKey(key)
	if err != nil {
		runtime.HandleError(err)
		return nil
	}

	m.stopFunc(clusterName)

	return nil
}

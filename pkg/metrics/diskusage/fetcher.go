// Package diskusage grabs pod disk usage from k8s
package diskusage

import (
	"fmt"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/cache"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . PodGetter

type PodGetter interface {
	Get(podName string) (*v1.Pod, error)
}

//counterfeiter:generate . NodeStatter

type NodeStatter interface {
	Summary(nodeName string) (NodeDiskUsage, error)
}

type Fetcher struct {
	nodeCache    *cache.Expiring
	nodeCacheTTL time.Duration
	podGetter    PodGetter
	nodeStatter  NodeStatter
}

func NewFetcher(nodeCache *cache.Expiring, nodeCacheTTL time.Duration, podGetter PodGetter, nodeStatter NodeStatter) *Fetcher {
	return &Fetcher{
		nodeCache:    nodeCache,
		nodeCacheTTL: nodeCacheTTL,
		podGetter:    podGetter,
		nodeStatter:  nodeStatter,
	}
}

func (f *Fetcher) DiskUsage(podName string) (int64, error) {
	pod, err := f.podGetter.Get(podName)
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve pod: %w", err)
	}

	var summary NodeDiskUsage
	cached, ok := f.nodeCache.Get(pod.Spec.NodeName)
	if ok {
		return calculatePodDiskUsage(podName, cached.(NodeDiskUsage))
	}

	summary, err = f.nodeStatter.Summary(pod.Spec.NodeName)
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve node summary: %w", err)
	}
	f.nodeCache.Set(pod.Spec.NodeName, summary, f.nodeCacheTTL)

	return calculatePodDiskUsage(podName, summary)
}

func calculatePodDiskUsage(podName string, summary NodeDiskUsage) (int64, error) {
	for _, pod := range summary.Pods {
		if pod.PodRef.Name == podName {
			var sum int64 = 0
			for _, container := range pod.Containers {
				if isIstio(container.Name) {
					continue
				}
				sum += container.RootFS.UsedBytes + container.Logs.UsedBytes
			}
			return sum, nil
		}
	}
	return 0, fmt.Errorf("disk usage for pod %q not found", podName)
}

func isIstio(containerName string) bool {
	return strings.HasPrefix(containerName, "istio-")
}

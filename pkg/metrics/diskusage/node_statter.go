package diskusage

import (
	"encoding/json"

	"k8s.io/client-go/rest"
)

type NodeDiskUsage struct {
	Pods []PodDiskUsage `json:"pods"`
}

type PodDiskUsage struct {
	PodRef     PodRef               `json:"podRef"`
	Containers []ContainerDiskUsage `json:"containers"`
}

type PodRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type ContainerDiskUsage struct {
	Name   string    `json:"name"`
	RootFS DiskUsage `json:"rootfs"`
	Logs   DiskUsage `json:"logs"`
}

type DiskUsage struct {
	UsedBytes int64 `json:"usedBytes"`
}

type nodeStatter struct {
	k8sRestClient rest.Interface
}

func NewNodeStatter(k8sRestClient rest.Interface) NodeStatter {
	return &nodeStatter{
		k8sRestClient: k8sRestClient,
	}
}

func (s *nodeStatter) Summary(nodeName string) (NodeDiskUsage, error) {
	result := s.k8sRestClient.
		Get().
		Resource("nodes").
		Name(nodeName).
		SubResource("proxy", "stats", "summary").
		Do()

	body, err := result.Raw()
	if err != nil {
		return NodeDiskUsage{}, err
	}

	var nodeDiskUsage NodeDiskUsage
	err = json.Unmarshal(body, &nodeDiskUsage)
	if err != nil {
		return NodeDiskUsage{}, err
	}
	return nodeDiskUsage, nil
}

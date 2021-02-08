package diskusage

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typesv1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type podGetter struct {
	podsClient typesv1.PodInterface
}

func NewPodGetter(podsClient typesv1.PodInterface) PodGetter {
	return &podGetter{podsClient: podsClient}
}

func (p *podGetter) Get(podName string) (*corev1.Pod, error) {
	return p.podsClient.Get(podName, metav1.GetOptions{})
}

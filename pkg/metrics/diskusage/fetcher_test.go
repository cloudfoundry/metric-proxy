package diskusage_test

import (
	"errors"
	"testing"
	"time"

	"code.cloudfoundry.org/metric-proxy/pkg/metrics/diskusage"
	"code.cloudfoundry.org/metric-proxy/pkg/metrics/diskusage/diskusagefakes"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/cache"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate k8s.io/apimachinery/pkg/util/clock.Clock

func TestDiskUsageFetcher(t *testing.T) {
	var (
		g                Gomega
		podGetter        *diskusagefakes.FakePodGetter
		returnedPod      *corev1.Pod
		returnedPodErr   error
		nodeStatter      *diskusagefakes.FakeNodeStatter
		returnedStats    diskusage.NodeDiskUsage
		returnedStatsErr error
		fetcher          *diskusage.Fetcher
		clock            *diskusagefakes.FakeClock
	)

	podResult := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-pod",
		},
		Spec: corev1.PodSpec{
			NodeName: "my-node",
		},
	}

	nodeResult := diskusage.NodeDiskUsage{
		Pods: []diskusage.PodDiskUsage{
			{
				PodRef: diskusage.PodRef{
					Name: "my-pod",
				},
				Containers: []diskusage.ContainerDiskUsage{
					{
						Name: "istio-init",
						RootFS: diskusage.DiskUsage{
							UsedBytes: 9999,
						},
						Logs: diskusage.DiskUsage{
							UsedBytes: 9999,
						},
					},
					{
						Name: "opi",
						RootFS: diskusage.DiskUsage{
							UsedBytes: 1000,
						},
						Logs: diskusage.DiskUsage{
							UsedBytes: 200,
						},
					},
					{
						Name: "opi-2",
						RootFS: diskusage.DiskUsage{
							UsedBytes: 30,
						},
						Logs: diskusage.DiskUsage{
							UsedBytes: 4,
						},
					},
				},
			},
		},
	}

	init := func() {
		returnedPod = nil
		returnedPodErr = nil
		returnedStats = diskusage.NodeDiskUsage{}
		returnedStatsErr = nil
	}

	setUp := func(t *testing.T) {
		g = NewGomegaWithT(t)

		podGetter = new(diskusagefakes.FakePodGetter)
		podGetter.GetReturns(returnedPod, returnedPodErr)

		nodeStatter = new(diskusagefakes.FakeNodeStatter)
		nodeStatter.SummaryReturns(returnedStats, returnedStatsErr)

		clock = new(diskusagefakes.FakeClock)
		nodeCache := cache.NewExpiringWithClock(clock)
		fetcher = diskusage.NewFetcher(nodeCache, time.Minute, podGetter, nodeStatter)
	}

	t.Run("it calculates pod disk usage", func(t *testing.T) {
		init()

		returnedPod = podResult
		returnedStats = nodeResult

		setUp(t)

		usage, err := fetcher.DiskUsage("my-pod")

		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(podGetter.GetCallCount()).To(Equal(1))
		g.Expect(podGetter.GetArgsForCall(0)).To(Equal("my-pod"))

		g.Expect(nodeStatter.SummaryCallCount()).To(Equal(1))
		g.Expect(nodeStatter.SummaryArgsForCall(0)).To(Equal("my-node"))

		g.Expect(usage).To(BeNumerically("==", 1234))
	})

	t.Run("cache is used when recent node summary is available", func(t *testing.T) {
		now := time.Now()
		init()

		returnedPod = podResult
		returnedStats = nodeResult

		setUp(t)

		clock.NowReturns(now)

		usage, err := fetcher.DiskUsage("my-pod")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(podGetter.GetCallCount()).To(Equal(1))
		g.Expect(nodeStatter.SummaryCallCount()).To(Equal(1))
		g.Expect(usage).To(BeNumerically("==", 1234))

		clock.NowReturns(now.Add(30 * time.Second))

		usage, err = fetcher.DiskUsage("my-pod")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(podGetter.GetCallCount()).To(Equal(2))
		g.Expect(nodeStatter.SummaryCallCount()).To(Equal(1))
		g.Expect(usage).To(BeNumerically("==", 1234))

		clock.NowReturns(time.Now().Add(2 * time.Minute))

		usage, err = fetcher.DiskUsage("my-pod")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(podGetter.GetCallCount()).To(Equal(3))
		g.Expect(nodeStatter.SummaryCallCount()).To(Equal(2))
		g.Expect(usage).To(BeNumerically("==", 1234))
	})

	t.Run("returning error when getting pod fails", func(t *testing.T) {
		init()
		returnedPodErr = errors.New("k8s problem")

		setUp(t)

		_, err := fetcher.DiskUsage("my-pod")
		g.Expect(err).To(MatchError(SatisfyAll(
			ContainSubstring("failed to retrieve pod"),
			ContainSubstring("k8s problem"),
		)))
	})

	t.Run("returning error when getting node stats fails", func(t *testing.T) {
		init()
		returnedPod = &corev1.Pod{
			Spec: corev1.PodSpec{
				NodeName: "my-node",
			},
		}
		returnedStatsErr = errors.New("k8s problem")

		setUp(t)

		_, err := fetcher.DiskUsage("my-pod")
		g.Expect(err).To(MatchError(SatisfyAll(
			ContainSubstring("failed to retrieve node summary"),
			ContainSubstring("k8s problem"),
		)))
	})

	t.Run("refreshing the cache when the pod metrics can't be found", func(t *testing.T) {
		init()
		returnedPod = podResult

		setUp(t)
		nodeStatter.SummaryReturnsOnCall(1, nodeResult, nil)

		// populate the node cache with empty results
		_, err := fetcher.DiskUsage("my-pod")
		g.Expect(err).To(HaveOccurred())

		usage, err := fetcher.DiskUsage("my-pod")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(usage).To(BeNumerically("==", 1234))
	})

	t.Run("when cache is refreshed for a missing pod, but pod still isn't found, it errors", func(t *testing.T) {
		init()
		returnedPod = podResult

		setUp(t)

		// populate the node cache with empty results
		_, err := fetcher.DiskUsage("my-pod")
		g.Expect(err).To(HaveOccurred())

		_, err = fetcher.DiskUsage("my-pod")
		g.Expect(err).To(MatchError(`disk usage for pod "my-pod" not found`))
	})

	t.Run("returning an error when the pod metrics don't exist", func(t *testing.T) {
		init()
		returnedPod = podResult

		setUp(t)

		_, err := fetcher.DiskUsage("my-pod")
		g.Expect(err).To(MatchError(`disk usage for pod "my-pod" not found`))
	})
}

/*
Copyright 2021 The Kubernetes Authors.

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

package e2enode

import (
	"fmt"
	"net"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/kubernetes/pkg/features"
	kubeletconfig "k8s.io/kubernetes/pkg/kubelet/apis/config"
	"k8s.io/kubernetes/test/e2e/common/node"
	"k8s.io/kubernetes/test/e2e/framework"
	imageutils "k8s.io/kubernetes/test/utils/image"

	"github.com/onsi/ginkgo"
)

const (
	defaultObservationTimeout = time.Minute * 4
)

var _ = SIGDescribe("GRPCProbe [Serial] [Disruptive]", func() {
	f := framework.NewDefaultFramework("grpc-probe-test")
	/*
		These tests are located here as they require tempSetCurrentKubeletConfig to enable the feature gate for gRPC probe.
		TODO: Once the feature gate has been removed, these tests should come back to test/e2e/common/container_probe.go.
	*/
	ginkgo.Context("when a container has a grpc probe", func() {
		tempSetCurrentKubeletConfig(f, func(initialConfig *kubeletconfig.KubeletConfiguration) {
			if initialConfig.FeatureGates == nil {
				initialConfig.FeatureGates = make(map[string]bool)
			}
			initialConfig.FeatureGates[string(features.GRPCContainerProbe)] = true
		})

		/*
			Release: v1.22
			Testname: Pod liveness probe, using grpc call, failure
			Description: A Pod is created with liveness probe on grpc service. Liveness probe on this endpoint will not fail. When liveness probe does not fail then the restart count MUST remain zero.
		*/
		ginkgo.It("should *not* be restarted with a GRPC liveness probe", func() {
			livenessProbe := &v1.Probe{
				Handler: v1.Handler{
					GRPC: &v1.GRPCAction{
						Port:    2379,
						Service: "",
						Host:    "127.0.0.1",
					},
				},
				InitialDelaySeconds: 15,
				FailureThreshold:    1,
			}
			pod := GRPCServerPodSpec(nil, livenessProbe, "etcd", 2379)
			node.RunLivenessTest(f, pod, 0, defaultObservationTimeout)
		})
	})
})

func GRPCServerPodSpec(readinessProbe, livenessProbe *v1.Probe, containerName string, port int) *v1.Pod {
	etcdLocalhostAddress := "127.0.0.1"
	if framework.TestContext.ClusterIsIPv6() {
		etcdLocalhostAddress = "::1"
	}
	etcdURL := fmt.Sprintf("http://%s", net.JoinHostPort(etcdLocalhostAddress, "2379"))
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test-grpc-" + string(uuid.NewUUID())},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  containerName,
					Image: imageutils.GetE2EImage(imageutils.Etcd),
					Command: []string{
						"/usr/local/bin/etcd",
						"--listen-client-urls",
						etcdURL,
						"--advertise-client-urls",
						etcdURL,
					},
					Ports:          []v1.ContainerPort{{ContainerPort: int32(port)}},
					LivenessProbe:  livenessProbe,
					ReadinessProbe: readinessProbe,
				},
			},
		},
	}
}

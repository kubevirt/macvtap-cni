// Copyright 2020 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tests_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

const (
	macvtapResource = "macvtap.network.kubevirt.io"
	networkResource = "k8s.v1.cni.cncf.io/networks"
	networkStatus   = "k8s.v1.cni.cncf.io/networks-status"
)

type reportedNetwork struct {
	Name      string            `json:"name"`
	Interface string            `json:"interface"`
	Mac       string            `json:"mac"`
	Dns       map[string]string `json:"dns"`
}

var postUrl = "/apis/k8s.cni.cncf.io/v1/namespaces/%s/network-attachment-definitions/%s"
var nad = `
	{
		"apiVersion":"k8s.cni.cncf.io/v1",
		"kind":"NetworkAttachmentDefinition",
		"metadata": {
			"name":"%s",
			"namespace":"%s",
			"annotations": {
				"k8s.v1.cni.cncf.io/resourceName": "%s"
			}
		},
		"spec":{
			"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"%s\", \"type\": \"macvtap\"}"
		}
	}
`

var _ = Describe("macvtap-cni", func() {

	lowerDevice := "eth0"
	namespace := "default"

	Describe("macvtap-cni infrastructure", func() {
		Context("WHEN make cluster-sync is executed", func() {
			It("THEN macvtap-cni daemonset is running in each k8s node", func() {
				daemonSetNamePrefix := "macvtap-cni"
				pods, _ := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
				nodes, _ := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
				Expect(len(pods.Items)).Should(BeNumerically(">", 0))
				Expect(len(nodes.Items)).Should(BeNumerically(">", 0))

				networkStatus := filterPods(pods.Items, func(pod v1.Pod) bool {
					return strings.HasPrefix(pod.Name, daemonSetNamePrefix)
				})
				Expect(len(networkStatus)).To(Equal(len(nodes.Items)))
			})

			It("THEN a macvtap custom network resource is exposed", func() {
				quantity := 100

				expectedResourceName := v1.ResourceName(buildMacvtapResourceName(lowerDevice))
				waitForNodeResourceAvailability(1*time.Minute, buildMacvtapResourceName(lowerDevice))

				nodes, _ := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
				for _, node := range nodes.Items {
					expectedQuantity, err := resource.ParseQuantity(strconv.Itoa(quantity))
					Expect(err).NotTo(HaveOccurred())

					// confirm capacity is OK
					macvtapCapacity := node.Status.Capacity[v1.ResourceName(expectedResourceName)]
					Expect(macvtapCapacity).To(Equal(expectedQuantity))

					// confirm allocatable is OK
					macvtapAllocatable := node.Status.Allocatable[v1.ResourceName(expectedResourceName)]
					Expect(macvtapAllocatable).To(Equal(expectedQuantity))
				}
			})
		})
	})

	Describe("macvtap resource creation", func() {

		Context("WHEN a macvtap interface is configured as a secondary interface", func() {
			networkAttachmentDefinitionName := "macvtap0"

			BeforeEach(func() {
				provisionNetworkAttachmentDefinition(networkAttachmentDefinitionName, lowerDevice, namespace)
			})

			AfterEach(func() {
				deleteNetworkAttachmentDefinition(networkAttachmentDefinitionName, namespace)
			})

			Context("WHEN a pod requests aforementioned macvtap resource", func() {
				podName := "megapod"
				containerName := "tinycontainer"

				container := v1.Container{
					Name:    containerName,
					Image:   "alpine",
					Command: []string{"/bin/sh", "-c", "sleep 999999"},
					Resources: v1.ResourceRequirements{
						Limits: buildMacvtapResourceRequest(lowerDevice, 1),
					},
				}

				pod1 := &v1.Pod{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Pod",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:        podName,
						Namespace:   namespace,
						Annotations: buildMacvtapNetworkAnnotations(networkAttachmentDefinitionName),
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{container},
					},
				}

				BeforeEach(func() {
					_, err := clientset.CoreV1().Pods(namespace).Create(context.TODO(), pod1, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())

					By("Waiting for pod to be ready")
					waitForPodReadiness(podName, namespace, 1*time.Minute)
				})

				AfterEach(func() {
					err := clientset.CoreV1().Pods(namespace).Delete(context.TODO(), podName, metav1.DeleteOptions{})
					Expect(err).NotTo(HaveOccurred())
				})

				It("SHOULD successfully get a second interface, of macvtap type, with configured MAC address", func() {
					pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					// assert resources have been allocated
					Expect(pod.Annotations).To(HaveKey(networkResource))
					Expect(pod.Spec.Containers).To(HaveLen(1))

					theContainer := pod.Spec.Containers[0]
					Expect(theContainer.Resources.Limits).To(Equal(buildMacvtapResourceRequest(lowerDevice, 1)))

					// assert MAC address is found on the second interface
					podNetworks := pod.Annotations[networkStatus]
					networks, err := parseNetwork(podNetworks)
					Expect(err).NotTo(HaveOccurred())
					Expect(networks).To(HaveLen(2))

					// macvtap iface is created as a secondary iface
					macvtapNetwork := networks[1]
					Expect(macvtapNetwork.Name).To(Equal(networkAttachmentDefinitionName))
					Expect(macvtapNetwork.Interface).To(Equal("net1"))
					Expect(macvtapNetwork.Mac).NotTo(BeEmpty())
				})
			})

			PContext("WHEN a VMI requests aforementioned macvtap resource", func() {
				It("THEN the macvtap interface can be used to reach the internet", func() {

				})

				Context("WHEN another VMI is created, having an additional macvtap interface", func() {
					It("THEN it can reach another VMI via the macvtap interface", func() {

					})

					It("THEN it cannot reach the host where its pod was scheduled", func() {

					})

					It("THEN it can reach the other hosts in the cluster", func() {

					})
				})
			})
		})
	})
})

func deleteNetworkAttachmentDefinition(macvtapIfaceName string, namespace string) rest.Result {
	return clientset.RESTClient().
		Delete().
		RequestURI(fmt.Sprintf(postUrl, namespace, macvtapIfaceName)).
		Do(context.TODO())
}

func provisionNetworkAttachmentDefinition(macvtapIfaceName string, lowerDeviceName string, namespace string) rest.Result {
	return clientset.RESTClient().
		Post().
		RequestURI(fmt.Sprintf(postUrl, namespace, macvtapIfaceName)).
		Body([]byte(fmt.Sprintf(nad, macvtapIfaceName, namespace, buildMacvtapResourceName(lowerDeviceName), macvtapIfaceName))).
		Do(context.TODO())
}

func filterPods(pods []v1.Pod, filterFunction func(v1.Pod) bool) []v1.Pod {
	filteredPods := make([]v1.Pod, 0)
	for _, pod := range pods {
		if filterFunction(pod) {
			filteredPods = append(filteredPods, pod)
		}
	}
	return filteredPods
}

func buildMacvtapResourceName(macvtapIfaceName string) string {
	return fmt.Sprintf("%s/%s", macvtapResource, macvtapIfaceName)
}

func waitForNodeResourceAvailability(timeout time.Duration, resourceName string) {
	checkForResourceAvailable := func() bool {
		nodeList, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())

		for _, node := range nodeList.Items {
			if _, ok := node.Status.Capacity[v1.ResourceName(resourceName)]; !ok {
				return false
			}
		}

		return true
	}

	Eventually(checkForResourceAvailable, timeout, 2*time.Second).Should(BeTrue())
}

func buildMacvtapNetworkAnnotations(networkName string) map[string]string {
	requestMacvtapNetwork := make(map[string]string)
	requestMacvtapNetwork[networkResource] = networkName
	return requestMacvtapNetwork
}

func buildMacvtapResourceRequest(resourceName string, quantity int) v1.ResourceList {
	return v1.ResourceList{
		v1.ResourceName(buildMacvtapResourceName(resourceName)): resource.MustParse(strconv.Itoa(quantity)),
	}
}

// Parse the json network reported by Multus in the networks-status annotations
func parseNetwork(network string) ([]reportedNetwork, error) {
	var reportedNetwork []reportedNetwork
	err := json.Unmarshal([]byte(network), &reportedNetwork)
	return reportedNetwork, err
}

func waitForPodReadiness(podName string, namespace string, timeout time.Duration) {
	isPodReady := func() bool {
		pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "Is able to GET a concreate POD")

		containerStatuses := pod.Status.ContainerStatuses
		readyContainers := 0
		for _, status := range containerStatuses {
			if status.Ready {
				readyContainers += 1
			}
		}

		return len(containerStatuses) > 0 && readyContainers == len(containerStatuses)
	}
	Eventually(isPodReady, timeout, 1*time.Second).Should(BeTrue())
}

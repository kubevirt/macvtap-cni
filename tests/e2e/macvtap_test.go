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
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("macvtap-cni", func() {

	namespace := "default"

	Describe("macvtap-cni infrastructure", func() {
		Context("WHEN make cluster-sync is executed", func() {
			It("THEN macvtap-cni daemonset is running in each k8s node", func() {
				daemonSetNamePrefix := "macvtap-cni"
				pods, _ := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{})
				nodes, _ := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
				Expect(len(pods.Items)).Should(BeNumerically(">", 0))
				Expect(len(nodes.Items)).Should(BeNumerically(">", 0))

				networkStatus := filterPods(pods.Items, func(pod v1.Pod) bool {
					return strings.HasPrefix(pod.Name, daemonSetNamePrefix)
				})
				Expect(len(networkStatus)).To(Equal(len(nodes.Items)))
			})
		})
	})

	Describe("macvtap resource creation", func() {

		Context("WHEN a lower device is configured accordingly", func() {
			BeforeEach(func() {

			})

			AfterEach(func() {

			})

			PIt("THEN a macvtap custom network resource is exposed", func() {

			})

			Context("WHEN a macvtap interface is configured as a secondary interface", func() {
				Context("WHEN a pod requests aforementioned macvtap resource", func() {
					PIt("THEN the pod successfully gets a second interface, of macvtap type, with configured MAC address", func() {

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
})

func filterPods(pods []v1.Pod, filterFunction func(v1.Pod) bool) []v1.Pod {
	filteredPods := make([]v1.Pod, 0)
	for _, pod := range pods {
		if filterFunction(pod) {
			filteredPods = append(filteredPods, pod)
		}
	}
	return filteredPods
}

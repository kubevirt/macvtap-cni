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

package tests

import (
	. "github.com/onsi/ginkgo"
)

var _ = Describe("macvtap-cni", func() {

	Describe("macvtap-cni infrastructure", func() {
		Context("WHEN make cluster-sync is executed", func() {
			PIt("THEN macvtap-cni daemonset is running in each k8s node", func() {

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


/*
Copyright 2025.

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

package controller

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
)

var _ = Describe("TalosClusterAddon Controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	var (
		addon            *talosv1alpha1.TalosClusterAddon
		addonName        string
		controlPlaneName string
		namespace        string
		ctx              context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		namespace = DefaultNamespace
		addonName = "test-addon-" + RandStringRunes(5)
		controlPlaneName = "test-cp-addon-" + RandStringRunes(5)

		// Create matching TalosControlPlane
		cp := &talosv1alpha1.TalosControlPlane{
			ObjectMeta: metav1.ObjectMeta{
				Name:      controlPlaneName,
				Namespace: namespace,
				Labels: map[string]string{
					"env": "production",
				},
			},
			Spec: talosv1alpha1.TalosControlPlaneSpec{
				Replicas:    1,
				Version:     "v1.10.4",
				KubeVersion: "v1.33.1",
				Mode:        "cloud",
			},
		}
		Expect(k8sClient.Create(ctx, cp)).To(Succeed())

		addon = &talosv1alpha1.TalosClusterAddon{
			ObjectMeta: metav1.ObjectMeta{
				Name:      addonName,
				Namespace: namespace,
			},
			Spec: talosv1alpha1.TalosClusterAddonSpec{
				ClusterSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"env": "production",
					},
				},
				HelmSpec: talosv1alpha1.HelmSpec{
					ChartName: "nginx",
					RepoURL:   "https://charts.bitnami.com/bitnami",
					Version:   "1.0.0",
				},
			},
		}
	})

	Context("When reconciling a TalosClusterAddon", func() {
		It("Should create TalosClusterAddonRelease for matching cluster", func() {
			By("Creating the TalosClusterAddon")
			Expect(k8sClient.Create(ctx, addon)).To(Succeed())

			By("Checking for TalosClusterAddonRelease creation")
			releaseName := fmt.Sprintf("%s-%s-addonrelease", controlPlaneName, addonName)
			release := &talosv1alpha1.TalosClusterAddonRelease{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: releaseName, Namespace: namespace}, release)
			}, timeout, interval).Should(Succeed())

			Expect(release.Spec.HelmSpec.ChartName).To(Equal("nginx"))
			Expect(release.Spec.ClusterRef.Name).To(Equal(controlPlaneName))

			By("Updating the Addon")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: addonName, Namespace: namespace}, addon)).To(Succeed())
				addon.Spec.HelmSpec.Version = "1.0.1"
				g.Expect(k8sClient.Update(ctx, addon)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			By("Verifying Release update")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: releaseName, Namespace: namespace}, release)).To(Succeed())
				g.Expect(release.Spec.HelmSpec.Version).To(Equal("1.0.1"))
			}, timeout, interval).Should(Succeed())
		})
	})
})

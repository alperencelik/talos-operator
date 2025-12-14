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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
)

var _ = Describe("TalosControlPlane Controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	var (
		talosControlPlane     *talosv1alpha1.TalosControlPlane
		talosControlPlaneName string
		namespace             string
		ctx                   context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		namespace = DefaultNamespace
		talosControlPlaneName = "test-tcp-" + RandStringRunes(5)

		talosControlPlane = &talosv1alpha1.TalosControlPlane{
			ObjectMeta: metav1.ObjectMeta{
				Name:      talosControlPlaneName,
				Namespace: namespace,
			},
			Spec: talosv1alpha1.TalosControlPlaneSpec{
				Replicas:    3,
				Version:     "v1.10.4",
				KubeVersion: "v1.33.1",
				Mode:        "cloud",
			},
		}
	})

	Context("When reconciling a TalosControlPlane", func() {
		It("Should successfully create the resource", func() {
			By("Creating the TalosControlPlane")
			Expect(k8sClient.Create(ctx, talosControlPlane)).To(Succeed())

			By("Checking for resource existence")
			createdResource := &talosv1alpha1.TalosControlPlane{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: talosControlPlaneName, Namespace: namespace}, createdResource)
			}, timeout, interval).Should(Succeed())

			Expect(createdResource.Spec.Replicas).To(Equal(int32(3)))
			Expect(createdResource.Spec.Mode).To(Equal("cloud"))
		})

		It("Should handle updates", func() {
			By("Creating the TalosControlPlane")
			Expect(k8sClient.Create(ctx, talosControlPlane)).To(Succeed())

			By("Updating the replicas")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: talosControlPlaneName, Namespace: namespace}, talosControlPlane)).To(Succeed())
				talosControlPlane.Spec.Replicas = 5
				g.Expect(k8sClient.Update(ctx, talosControlPlane)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			By("Verifying the update")
			createdResource := &talosv1alpha1.TalosControlPlane{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: talosControlPlaneName, Namespace: namespace}, createdResource)).To(Succeed())
				g.Expect(createdResource.Spec.Replicas).To(Equal(int32(5)))
			}, timeout, interval).Should(Succeed())
		})

		It("Should handle deletion", func() {
			By("Creating the TalosControlPlane")
			Expect(k8sClient.Create(ctx, talosControlPlane)).To(Succeed())

			By("Deleting the TalosControlPlane")
			Expect(k8sClient.Delete(ctx, talosControlPlane)).To(Succeed())

			By("Verifying resource is deleted")
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: talosControlPlaneName, Namespace: namespace}, talosControlPlane)
			}, timeout, interval).ShouldNot(Succeed())
		})
	})
})

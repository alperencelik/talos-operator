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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
)

var _ = Describe("TalosMachine Controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	var (
		talosMachine     *talosv1alpha1.TalosMachine
		talosMachineName string
		namespace        string
		ctx              context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		namespace = DefaultNamespace
		talosMachineName = "test-machine-" + RandStringRunes(5)

		talosMachine = &talosv1alpha1.TalosMachine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      talosMachineName,
				Namespace: namespace,
			},
			Spec: talosv1alpha1.TalosMachineSpec{
				Endpoint:       testMachineIP,
				Version:        testTalosVersion,
				DeletionPolicy: testDeletionPolicyReset,
			},
		}
	})

	Context("When reconciling a TalosMachine", func() {
		It("Should successfully create the resource", func() {
			By("Creating the TalosMachine")
			Expect(k8sClient.Create(ctx, talosMachine)).To(Succeed())

			By("Checking for resource existence")
			createdResource := &talosv1alpha1.TalosMachine{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: talosMachineName, Namespace: namespace}, createdResource)
			}, timeout, interval).Should(Succeed())

			Expect(createdResource.Spec.Endpoint).To(Equal(testMachineIP))
			Expect(createdResource.Spec.Version).To(Equal(testTalosVersion))
		})

		It("Should handle updates", func() {
			By("Creating the TalosMachine")
			Expect(k8sClient.Create(ctx, talosMachine)).To(Succeed())

			By("Updating the version")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: talosMachineName, Namespace: namespace}, talosMachine)).To(Succeed())
				talosMachine.Spec.Version = "v1.5.1"
				g.Expect(k8sClient.Update(ctx, talosMachine)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			By("Verifying the update")
			createdResource := &talosv1alpha1.TalosMachine{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: talosMachineName, Namespace: namespace}, createdResource)).To(Succeed())
				g.Expect(createdResource.Spec.Version).To(Equal("v1.5.1"))
			}, timeout, interval).Should(Succeed())
		})

		It("Should handle deletion", func() {
			By("Creating the TalosMachine")
			Expect(k8sClient.Create(ctx, talosMachine)).To(Succeed())

			By("Deleting the TalosMachine")
			Expect(k8sClient.Delete(ctx, talosMachine)).To(Succeed())

			By("Verifying resource is deleted")
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: talosMachineName, Namespace: namespace}, talosMachine)
			}, timeout, interval).ShouldNot(Succeed())
		})
	})

	Context("When reconciling a TalosMachine in DryRun mode", func() {
		It("Should emit DryRun events and not mutate the status", func() {
			By("Creating the TalosMachine with the DryRun annotation")
			talosMachine.Annotations = map[string]string{
				ReconcileModeAnnotation: ReconcileModeDryRun,
			}
			Expect(k8sClient.Create(ctx, talosMachine)).To(Succeed())

			By("Waiting for a DryRun event to be recorded")
			Eventually(func(g Gomega) {
				var eventList corev1.EventList
				g.Expect(k8sClient.List(ctx, &eventList, client.InNamespace(namespace))).To(Succeed())
				var found bool
				for _, e := range eventList.Items {
					if e.InvolvedObject.Name == talosMachineName && e.Reason == EventReasonDryRun {
						found = true
						break
					}
				}
				g.Expect(found).To(BeTrue(), "expected a DryRun event for the TalosMachine")
			}, timeout, interval).Should(Succeed())

			By("Verifying the status is not mutated")
			Consistently(func(g Gomega) {
				fetched := &talosv1alpha1.TalosMachine{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: talosMachineName, Namespace: namespace}, fetched)).To(Succeed())
				g.Expect(fetched.Status.State).To(BeEmpty())
				g.Expect(fetched.Status.Config).To(BeEmpty())
			}, time.Second*2, interval).Should(Succeed())
		})
	})
})

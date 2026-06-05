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
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
				Replicas:       3,
				Version:        testTalosVersion,
				KubeVersion:    testKubeVersion,
				Mode:           testModeCloud,
				DeletionPolicy: testDeletionPolicyReset,
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
			Expect(createdResource.Spec.Mode).To(Equal(testModeCloud))
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

	Context("When reconciling a TalosControlPlane in DryRun mode", func() {
		It("Should skip reconciliation entirely in container mode", func() {
			By("Creating a container-mode TalosControlPlane with the DryRun annotation")
			talosControlPlane.Annotations = map[string]string{
				ReconcileModeAnnotation: ReconcileModeDryRun,
			}
			talosControlPlane.Spec.Mode = TalosModeContainer
			Expect(k8sClient.Create(ctx, talosControlPlane)).To(Succeed())

			By("Verifying nothing is reconciled: no DryRun event, no resources, no status")
			Consistently(func(g Gomega) {
				// No DryRun event is recorded since container mode skips before the simulation starts
				var eventList corev1.EventList
				g.Expect(k8sClient.List(ctx, &eventList, client.InNamespace(namespace))).To(Succeed())
				for _, e := range eventList.Items {
					g.Expect(e.InvolvedObject.Name == talosControlPlaneName && e.Reason == "DryRun").To(BeFalse(),
						"expected no DryRun event for a container-mode TalosControlPlane")
				}
				// No config ConfigMap is created
				cm := &corev1.ConfigMap{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: talosControlPlaneName + "-config", Namespace: namespace}, cm)).NotTo(Succeed())
				// Status stays untouched
				fetched := &talosv1alpha1.TalosControlPlane{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: talosControlPlaneName, Namespace: namespace}, fetched)).To(Succeed())
				g.Expect(fetched.Status.State).To(BeEmpty())
				g.Expect(fetched.Status.Config).To(BeEmpty())
				g.Expect(fetched.Status.SecretBundle).To(BeEmpty())
			}, time.Second*3, interval).Should(Succeed())
		})

		It("Should simulate without persisting in metal mode", func() {
			By("Creating a metal-mode TalosControlPlane with the DryRun annotation")
			talosControlPlane.Annotations = map[string]string{
				ReconcileModeAnnotation: ReconcileModeDryRun,
			}
			talosControlPlane.Spec.Mode = TalosModeMetal
			talosControlPlane.Spec.Endpoint = "https://" + testMachineIP + ":6443"
			talosControlPlane.Spec.MetalSpec = talosv1alpha1.MetalSpec{
				Machines: []talosv1alpha1.Machine{
					{Address: ptr.To(testMachineIP)},
				},
			}
			Expect(k8sClient.Create(ctx, talosControlPlane)).To(Succeed())

			By("Waiting for a DryRun event to be recorded")
			Eventually(func(g Gomega) {
				var eventList corev1.EventList
				g.Expect(k8sClient.List(ctx, &eventList, client.InNamespace(namespace))).To(Succeed())
				var found bool
				for _, e := range eventList.Items {
					if e.InvolvedObject.Name == talosControlPlaneName && e.Reason == "DryRun" {
						found = true
						break
					}
				}
				g.Expect(found).To(BeTrue(), "expected a DryRun event for the TalosControlPlane")
			}, timeout, interval).Should(Succeed())

			By("Verifying nothing is persisted")
			Consistently(func(g Gomega) {
				// No TalosMachine children are created
				machines := &talosv1alpha1.TalosMachineList{}
				g.Expect(k8sClient.List(ctx, machines, client.InNamespace(namespace))).To(Succeed())
				for _, tm := range machines.Items {
					if tm.Spec.ControlPlaneRef != nil {
						g.Expect(tm.Spec.ControlPlaneRef.Name).NotTo(Equal(talosControlPlaneName),
							"expected no TalosMachine children for a DryRun TalosControlPlane")
					}
				}
				// No config ConfigMap is created
				cm := &corev1.ConfigMap{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: talosControlPlaneName + "-config", Namespace: namespace}, cm)).NotTo(Succeed())
				// Status stays untouched
				fetched := &talosv1alpha1.TalosControlPlane{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: talosControlPlaneName, Namespace: namespace}, fetched)).To(Succeed())
				g.Expect(fetched.Status.State).To(BeEmpty())
				g.Expect(fetched.Status.Config).To(BeEmpty())
				g.Expect(fetched.Status.SecretBundle).To(BeEmpty())
			}, time.Second*3, interval).Should(Succeed())
		})
	})
})

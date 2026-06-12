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

var _ = Describe("TalosWorker Controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	var (
		talosWorker      *talosv1alpha1.TalosWorker
		talosWorkerName  string
		controlPlaneName string
		namespace        string
		ctx              context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		namespace = DefaultNamespace
		talosWorkerName = "test-worker-" + RandStringRunes(5)
		controlPlaneName = "test-cp-for-worker-" + RandStringRunes(5)

		// Create a dummy ControlPlane for reference
		cp := &talosv1alpha1.TalosControlPlane{
			ObjectMeta: metav1.ObjectMeta{
				Name:      controlPlaneName,
				Namespace: namespace,
			},
			Spec: talosv1alpha1.TalosControlPlaneSpec{
				Replicas:       1,
				Version:        testTalosVersion,
				KubeVersion:    testKubeVersion,
				Mode:           testModeCloud,
				DeletionPolicy: testDeletionPolicyReset,
			},
		}
		Expect(k8sClient.Create(ctx, cp)).To(Succeed())

		talosWorker = &talosv1alpha1.TalosWorker{
			ObjectMeta: metav1.ObjectMeta{
				Name:      talosWorkerName,
				Namespace: namespace,
			},
			Spec: talosv1alpha1.TalosWorkerSpec{
				Replicas:       3,
				Version:        testTalosVersion,
				KubeVersion:    testKubeVersion,
				Mode:           testModeCloud,
				DeletionPolicy: testDeletionPolicyReset,
				ControlPlaneRef: corev1.LocalObjectReference{
					Name: controlPlaneName,
				},
			},
		}
	})

	Context("When reconciling a TalosWorker", func() {
		It("Should successfully create the resource", func() {
			By("Creating the TalosWorker")
			Expect(k8sClient.Create(ctx, talosWorker)).To(Succeed())

			By("Checking for resource existence")
			createdResource := &talosv1alpha1.TalosWorker{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: talosWorkerName, Namespace: namespace}, createdResource)
			}, timeout, interval).Should(Succeed())

			Expect(createdResource.Spec.Replicas).To(Equal(int32(3)))
			Expect(createdResource.Spec.ControlPlaneRef.Name).To(Equal(controlPlaneName))
		})

		It("Should handle updates", func() {
			By("Creating the TalosWorker")
			Expect(k8sClient.Create(ctx, talosWorker)).To(Succeed())

			By("Updating the replicas")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: talosWorkerName, Namespace: namespace}, talosWorker)).To(Succeed())
				talosWorker.Spec.Replicas = 4
				g.Expect(k8sClient.Update(ctx, talosWorker)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			By("Verifying the update")
			createdResource := &talosv1alpha1.TalosWorker{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: talosWorkerName, Namespace: namespace}, createdResource)).To(Succeed())
				g.Expect(createdResource.Spec.Replicas).To(Equal(int32(4)))
			}, timeout, interval).Should(Succeed())
		})

		It("Should handle deletion", func() {
			By("Creating the TalosWorker")
			Expect(k8sClient.Create(ctx, talosWorker)).To(Succeed())

			By("Deleting the TalosWorker")
			Expect(k8sClient.Delete(ctx, talosWorker)).To(Succeed())

			By("Verifying resource is deleted")
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: talosWorkerName, Namespace: namespace}, talosWorker)
			}, timeout, interval).ShouldNot(Succeed())
		})
	})

	Context("When reconciling a TalosWorker in DryRun mode", func() {
		It("Should skip reconciliation entirely in container mode", func() {
			By("Stubbing the TalosControlPlane SecretBundle so the worker passes the precondition")
			Eventually(func(g Gomega) {
				cp := &talosv1alpha1.TalosControlPlane{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: controlPlaneName, Namespace: namespace}, cp)).To(Succeed())
				cp.Status.SecretBundle = "stub-secret-bundle"
				g.Expect(k8sClient.Status().Update(ctx, cp)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			By("Creating a container-mode TalosWorker with the DryRun annotation")
			talosWorker.Annotations = map[string]string{
				ReconcileModeAnnotation: ReconcileModeDryRun,
			}
			talosWorker.Spec.Mode = TalosModeContainer
			Expect(k8sClient.Create(ctx, talosWorker)).To(Succeed())

			By("Verifying nothing is reconciled: no DryRun event, no resources, no status")
			Consistently(func(g Gomega) {
				// No DryRun event is recorded since container mode skips before the simulation starts
				var eventList corev1.EventList
				g.Expect(k8sClient.List(ctx, &eventList, client.InNamespace(namespace))).To(Succeed())
				for _, e := range eventList.Items {
					g.Expect(e.InvolvedObject.Name == talosWorkerName && e.Reason == EventReasonDryRun).To(BeFalse(),
						"expected no DryRun event for a container-mode TalosWorker")
				}
				// No worker config ConfigMap is created
				cm := &corev1.ConfigMap{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: talosWorkerName + "-config", Namespace: namespace}, cm)).NotTo(Succeed())
				// Status stays untouched
				fetched := &talosv1alpha1.TalosWorker{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: talosWorkerName, Namespace: namespace}, fetched)).To(Succeed())
				g.Expect(fetched.Status.State).To(BeEmpty())
				g.Expect(fetched.Status.Config).To(BeEmpty())
			}, time.Second*3, interval).Should(Succeed())
		})
	})
})

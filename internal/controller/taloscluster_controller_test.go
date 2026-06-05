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
	"math/rand"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
)

var _ = Describe("TalosCluster Controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	var (
		talosCluster     *talosv1alpha1.TalosCluster
		talosClusterName string
		namespace        string
		ctx              context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		namespace = DefaultNamespace
		talosClusterName = "test-talos-cluster-" + RandStringRunes(5)

		talosCluster = &talosv1alpha1.TalosCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      talosClusterName,
				Namespace: namespace,
			},
			Spec: talosv1alpha1.TalosClusterSpec{
				ControlPlane: &talosv1alpha1.TalosControlPlaneSpec{
					Replicas:       3,
					Version:        testTalosVersion,
					KubeVersion:    "v1.31.0",
					Mode:           testModeCloud,
					DeletionPolicy: testDeletionPolicyReset,
				},
				Worker: &talosv1alpha1.TalosWorkerSpec{
					Replicas:       3,
					Version:        testTalosVersion,
					KubeVersion:    "v1.31.0",
					Mode:           testModeCloud,
					DeletionPolicy: testDeletionPolicyReset,
				},
			},
		}
	})

	Context("When reconciling a TalosCluster", func() {
		It("Should create TalosControlPlane and TalosWorker resources", func() {
			By("Creating the TalosCluster")
			Expect(k8sClient.Create(ctx, talosCluster)).To(Succeed())

			By("Checking for TalosControlPlane creation")
			createdControlPlane := &talosv1alpha1.TalosControlPlane{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: talosClusterName, Namespace: namespace}, createdControlPlane)
			}, timeout, interval).Should(Succeed())

			Expect(createdControlPlane.Spec.Replicas).To(Equal(int32(3)))
			Expect(createdControlPlane.Spec.Version).To(Equal(testTalosVersion))

			By("Checking for TalosWorker creation")
			createdWorker := &talosv1alpha1.TalosWorker{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: talosClusterName, Namespace: namespace}, createdWorker)
			}, timeout, interval).Should(Succeed())

			Expect(createdWorker.Spec.Replicas).To(Equal(int32(3)))
			Expect(createdWorker.Spec.ControlPlaneRef.Name).To(Equal(talosClusterName))

			By("Verifying OwnerReferences")
			// Helper to check owner reference
			isOwnedBy := func(obj client.Object, owner *talosv1alpha1.TalosCluster) bool {
				for _, ref := range obj.GetOwnerReferences() {
					if ref.UID == owner.UID {
						return true
					}
				}
				return false
			}
			Expect(isOwnedBy(createdControlPlane, talosCluster)).To(BeTrue(), "TalosControlPlane should be owned by TalosCluster")
			Expect(isOwnedBy(createdWorker, talosCluster)).To(BeTrue(), "TalosWorker should be owned by TalosCluster")

			By("Checking Finalizer")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: talosClusterName, Namespace: namespace}, talosCluster)).To(Succeed())
				g.Expect(talosCluster.Finalizers).To(ContainElement(talosv1alpha1.TalosClusterFinalizer))
			}, timeout, interval).Should(Succeed())
		})

		It("Should update child resources when TalosCluster is updated", func() {
			By("Creating the TalosCluster")
			Expect(k8sClient.Create(ctx, talosCluster)).To(Succeed())

			By("Waiting for child resources")
			createdControlPlane := &talosv1alpha1.TalosControlPlane{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: talosClusterName, Namespace: namespace}, createdControlPlane)
			}, timeout, interval).Should(Succeed())

			By("Updating TalosCluster spec")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: talosClusterName, Namespace: namespace}, talosCluster)).To(Succeed())
				talosCluster.Spec.ControlPlane.Replicas = 5
				g.Expect(k8sClient.Update(ctx, talosCluster)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			By("Verifying TalosControlPlane update")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: talosClusterName, Namespace: namespace}, createdControlPlane)).To(Succeed())
				g.Expect(createdControlPlane.Spec.Replicas).To(Equal(int32(5)))
			}, timeout, interval).Should(Succeed())
		})

		It("Should not create child resources in DryRun mode", func() {
			By("Creating the TalosCluster with the DryRun annotation")
			talosCluster.Annotations = map[string]string{
				ReconcileModeAnnotation: ReconcileModeDryRun,
			}
			Expect(k8sClient.Create(ctx, talosCluster)).To(Succeed())

			By("Waiting for the finalizer to be added (finalizer management runs normally)")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: talosClusterName, Namespace: namespace}, talosCluster)).To(Succeed())
				g.Expect(talosCluster.Finalizers).To(ContainElement(talosv1alpha1.TalosClusterFinalizer))
			}, timeout, interval).Should(Succeed())

			By("Waiting for a DryRun event to be recorded")
			Eventually(func(g Gomega) {
				var eventList corev1.EventList
				g.Expect(k8sClient.List(ctx, &eventList, client.InNamespace(namespace))).To(Succeed())
				var found bool
				for _, e := range eventList.Items {
					if e.InvolvedObject.Name == talosClusterName && e.Reason == "DryRun" {
						found = true
						break
					}
				}
				g.Expect(found).To(BeTrue(), "expected a DryRun event for the TalosCluster")
			}, timeout, interval).Should(Succeed())

			By("Verifying no child resources are created")
			Consistently(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: talosClusterName, Namespace: namespace}, &talosv1alpha1.TalosControlPlane{})).NotTo(Succeed())
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: talosClusterName, Namespace: namespace}, &talosv1alpha1.TalosWorker{})).NotTo(Succeed())
			}, time.Second*3, interval).Should(Succeed())
		})

		It("Should handle deletion correctly", func() {
			By("Creating the TalosCluster")
			Expect(k8sClient.Create(ctx, talosCluster)).To(Succeed())

			By("Waiting for finalizer")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: talosClusterName, Namespace: namespace}, talosCluster)).To(Succeed())
				g.Expect(talosCluster.Finalizers).To(ContainElement(talosv1alpha1.TalosClusterFinalizer))
			}, timeout, interval).Should(Succeed())

			By("Deleting the TalosCluster")
			Expect(k8sClient.Delete(ctx, talosCluster)).To(Succeed())

			By("Verifying resource is deleted")
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: talosClusterName, Namespace: namespace}, talosCluster)
			}, timeout, interval).ShouldNot(Succeed())
			// Also verify child resources are deleted
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: talosClusterName, Namespace: namespace}, &talosv1alpha1.TalosControlPlane{})
			}, timeout, interval).ShouldNot(Succeed())

			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: talosClusterName, Namespace: namespace}, &talosv1alpha1.TalosWorker{})
			}, timeout, interval).ShouldNot(Succeed())

		})
	})
})

func RandStringRunes(n int) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz1234567890")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

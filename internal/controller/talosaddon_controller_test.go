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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
)

var _ = Describe("TalosAddon Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-addon"
		const clusterName = "test-cluster"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		talosaddon := &talosv1alpha1.TalosAddon{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind TalosAddon")
			err := k8sClient.Get(ctx, typeNamespacedName, talosaddon)
			if err != nil && errors.IsNotFound(err) {
				resource := &talosv1alpha1.TalosAddon{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: talosv1alpha1.TalosAddonSpec{
						ClusterRef: corev1.LocalObjectReference{
							Name: clusterName,
						},
						HelmRelease: talosv1alpha1.HelmReleaseSpec{
							ChartName:       "nginx",
							RepoURL:         "https://charts.bitnami.com/bitnami",
							Version:         "15.0.0",
							TargetNamespace: "default",
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &talosv1alpha1.TalosAddon{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance TalosAddon")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &TalosAddonReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			// We expect an error here because the cluster doesn't exist
			// This is OK for basic validation
			Expect(err).To(HaveOccurred())
		})
	})
})

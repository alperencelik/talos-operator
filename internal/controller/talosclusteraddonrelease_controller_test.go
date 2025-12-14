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

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
)

var _ = Describe("TalosClusterAddonRelease Controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	var (
		release          *talosv1alpha1.TalosClusterAddonRelease
		releaseName      string
		controlPlaneName string
		namespace        string
		ctx              context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		namespace = "default"
		releaseName = "test-release-" + RandStringRunes(5)
		controlPlaneName = "test-cp-release-" + RandStringRunes(5)

		// Create matching TalosControlPlane
		cp := &talosv1alpha1.TalosControlPlane{
			ObjectMeta: metav1.ObjectMeta{
				Name:      controlPlaneName,
				Namespace: namespace,
			},
			Spec: talosv1alpha1.TalosControlPlaneSpec{
				Replicas:    1,
				Version:     "v1.10.4",
				KubeVersion: "v1.33.1",
				Mode:        "cloud",
			},
		}
		Expect(k8sClient.Create(ctx, cp)).To(Succeed())

		// Create kubeconfig secret
		kubeconfigSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      controlPlaneName + "-kubeconfig",
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"kubeconfig": []byte("dummy-kubeconfig"),
			},
		}
		Expect(k8sClient.Create(ctx, kubeconfigSecret)).To(Succeed())

		release = &talosv1alpha1.TalosClusterAddonRelease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      releaseName,
				Namespace: namespace,
			},
			Spec: talosv1alpha1.TalosClusterAddonReleaseSpec{
				ClusterRef: corev1.ObjectReference{
					Name:       controlPlaneName,
					Namespace:  namespace,
					Kind:       "TalosControlPlane",
					APIVersion: talosv1alpha1.GroupVersion.String(),
				},
				HelmSpec: talosv1alpha1.HelmSpec{
					ChartName:        "nginx",
					RepoURL:          "https://charts.bitnami.com/bitnami",
					Version:          "1.0.0",
					ReleaseName:      "my-nginx",
					ReleaseNamespace: "default",
				},
			},
		}
	})

	Context("When reconciling a TalosClusterAddonRelease", func() {
		It("Should successfully reconcile and add finalizer", func() {
			By("Creating the TalosClusterAddonRelease")
			Expect(k8sClient.Create(ctx, release)).To(Succeed())

			By("Checking for Finalizer")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: releaseName, Namespace: namespace}, release)).To(Succeed())
				g.Expect(release.Finalizers).To(ContainElement(talosv1alpha1.TalosClusterAddonReleaseFinalizer))
			}, timeout, interval).Should(Succeed())

			// Likely fails later due to bad kubeconfig, but Finalizer presence means controller started work.
		})
	})
})

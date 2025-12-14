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

var _ = Describe("TalosEtcdBackup Controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	var (
		talosEtcdBackup     *talosv1alpha1.TalosEtcdBackup
		talosEtcdBackupName string
		controlPlaneName    string
		namespace           string
		ctx                 context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		namespace = "default"
		talosEtcdBackupName = "test-backup-" + RandStringRunes(5)
		controlPlaneName = "test-cp-backup-" + RandStringRunes(5)

		// Create dummy ControlPlane
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

		// Create dummy Secrets for S3
		secretName := "s3-secret-" + RandStringRunes(5)
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: namespace,
			},
			StringData: map[string]string{
				"access-key": "dummy",
				"secret-key": "dummy",
			},
		}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		talosEtcdBackup = &talosv1alpha1.TalosEtcdBackup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      talosEtcdBackupName,
				Namespace: namespace,
			},
			Spec: talosv1alpha1.TalosEtcdBackupSpec{
				TalosControlPlaneRef: &corev1.LocalObjectReference{
					Name: controlPlaneName,
				},
				BackupStorage: talosv1alpha1.BackupStorage{
					S3: &talosv1alpha1.S3Storage{
						Bucket: "test-bucket",
						Region: "us-east-1",
						AccessKeyID: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: secretName,
							},
							Key: "access-key",
						},
						SecretAccessKey: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: secretName,
							},
							Key: "secret-key",
						},
					},
				},
			},
		}
	})

	Context("When reconciling a TalosEtcdBackup", func() {
		It("Should successfully create the resource and add finalizer", func() {
			By("Creating the TalosEtcdBackup")
			Expect(k8sClient.Create(ctx, talosEtcdBackup)).To(Succeed())

			By("Checking for Finalizer")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: talosEtcdBackupName, Namespace: namespace}, talosEtcdBackup)).To(Succeed())
				g.Expect(talosEtcdBackup.Finalizers).To(ContainElement(talosv1alpha1.TalosEtcdBackupFinalizer))
			}, timeout, interval).Should(Succeed())
		})
		It("Should handle deletion and remove finalizer", func() {
			By("Creating the TalosEtcdBackup")
			Expect(k8sClient.Create(ctx, talosEtcdBackup)).To(Succeed())

			By("Deleting the TalosEtcdBackup")
			Expect(k8sClient.Delete(ctx, talosEtcdBackup)).To(Succeed())

			By("Verifying resource is deleted")
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: talosEtcdBackupName, Namespace: namespace}, talosEtcdBackup)
			}, timeout, interval).ShouldNot(Succeed())
		})
	})

	// TODO: Find a way to mock S3 interactions to test backup process
	// backupKey := storage.GenerateBackupKey(controlPlaneName)
	// By("Checking for backup process initiation")
	// Eventually(func(g Gomega) {
	// g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: talosEtcdBackupName, Namespace: namespace}, talosEtcdBackup)).To(Succeed())
	// g.Expect(talosEtcdBackup.Status.FilePath).To(Equal(backupKey))
	// }, timeout, interval).Should(Succeed())
})

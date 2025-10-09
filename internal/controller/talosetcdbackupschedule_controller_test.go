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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("TalosEtcdBackupSchedule Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-backup-schedule"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		talosetcdbackupschedule := &talosv1alpha1.TalosEtcdBackupSchedule{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind TalosEtcdBackupSchedule")
			err := k8sClient.Get(ctx, typeNamespacedName, talosetcdbackupschedule)
			if err != nil && errors.IsNotFound(err) {
				retention := int32(3)
				resource := &talosv1alpha1.TalosEtcdBackupSchedule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: talosv1alpha1.TalosEtcdBackupScheduleSpec{
						Schedule:  "0 2 * * *",
						Retention: &retention,
						BackupTemplate: talosv1alpha1.TalosEtcdBackupTemplateSpec{
							Spec: talosv1alpha1.TalosEtcdBackupSpec{
								TalosControlPlaneRef: &corev1.LocalObjectReference{
									Name: "test-controlplane",
								},
								BackupStorage: talosv1alpha1.BackupStorage{
									S3: &talosv1alpha1.S3Storage{
										Bucket: "test-bucket",
										Region: "us-west-2",
										AccessKeyID: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "test-secret",
											},
											Key: "accessKeyID",
										},
										SecretAccessKey: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "test-secret",
											},
											Key: "secretAccessKey",
										},
									},
								},
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &talosv1alpha1.TalosEtcdBackupSchedule{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance TalosEtcdBackupSchedule")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &TalosEtcdBackupScheduleReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

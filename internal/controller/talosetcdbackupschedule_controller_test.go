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

var _ = Describe("TalosEtcdBackupSchedule Controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	var (
		schedule     *talosv1alpha1.TalosEtcdBackupSchedule
		scheduleName string
		namespace    string
		ctx          context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		namespace = DefaultNamespace
		scheduleName = "test-schedule-" + RandStringRunes(5)

		schedule = &talosv1alpha1.TalosEtcdBackupSchedule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      scheduleName,
				Namespace: namespace,
			},
			Spec: talosv1alpha1.TalosEtcdBackupScheduleSpec{
				Schedule: "*/1 * * * *", // Every minute
				BackupTemplate: talosv1alpha1.TalosEtcdBackupTemplateSpec{
					Spec: talosv1alpha1.TalosEtcdBackupSpec{
						TalosControlPlaneRef: &corev1.LocalObjectReference{
							Name: "some-cp",
						},
						BackupStorage: talosv1alpha1.BackupStorage{
							S3: &talosv1alpha1.S3Storage{
								Bucket: "bucket",
								Region: "us-east-1",
								AccessKeyID: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{Name: "s"}, Key: "k",
								},
								SecretAccessKey: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{Name: "s"}, Key: "k",
								},
							},
						},
					},
				},
			},
		}
	})

	Context("When reconciling a TalosEtcdBackupSchedule", func() {
		It("Should successfully create the schedule and trigger an initial backup", func() {
			By("Creating the TalosEtcdBackupSchedule")
			Expect(k8sClient.Create(ctx, schedule)).To(Succeed())

			By("Checking for Finalizer and NextScheduleTime")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: scheduleName, Namespace: namespace}, schedule)).To(Succeed())
				g.Expect(schedule.Finalizers).To(ContainElement(talosv1alpha1.TalosEtcdBackupScheduleFinalizer))
				g.Expect(schedule.Status.NextScheduleTime).NotTo(BeNil())
			}, timeout, interval).Should(Succeed())

			By("Checking for created TalosEtcdBackup")
			Eventually(func(g Gomega) {
				var backupList talosv1alpha1.TalosEtcdBackupList
				g.Expect(k8sClient.List(ctx, &backupList, client.InNamespace(namespace), client.MatchingLabels{
					talosv1alpha1.TalosEtcdBackupScheduleLabelKey: scheduleName,
				})).To(Succeed())
				g.Expect(backupList.Items).To(HaveLen(1))
			}, timeout, interval).Should(Succeed())
		})
		It("Should handle deletion properly", func() {
			By("Creating the TalosEtcdBackupSchedule")
			Expect(k8sClient.Create(ctx, schedule)).To(Succeed())

			By("Deleting the TalosEtcdBackupSchedule")
			Expect(k8sClient.Delete(ctx, schedule)).To(Succeed())

			By("Verifying resource is deleted")
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: scheduleName, Namespace: namespace}, schedule)
			}, timeout, interval).ShouldNot(Succeed())
			// Also verify the child backups are deleted if any
			By("Verifying associated TalosEtcdBackups are deleted")
			Eventually(func(g Gomega) {
				var backupList talosv1alpha1.TalosEtcdBackupList
				g.Expect(k8sClient.List(ctx, &backupList, client.InNamespace(namespace), client.MatchingLabels{
					talosv1alpha1.TalosEtcdBackupScheduleLabelKey: scheduleName,
				})).To(Succeed())
				g.Expect(backupList.Items).To(BeEmpty())
			}, timeout, interval).Should(Succeed())
		})
	})
})

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

package v1alpha1

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
	// TODO (user): Add any additional imports if needed
)

var _ = Describe("TalosControlPlane Webhook", func() {
	var (
		ctx       context.Context
		obj       *talosv1alpha1.TalosControlPlane
		oldObj    *talosv1alpha1.TalosControlPlane
		validator TalosControlPlaneCustomValidator
	)

	BeforeEach(func() {
		ctx = context.Background()
		obj = &talosv1alpha1.TalosControlPlane{}
		oldObj = &talosv1alpha1.TalosControlPlane{}
		validator = TalosControlPlaneCustomValidator{}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
		// TODO (user): Add any setup logic common to all tests
	})

	AfterEach(func() {
		// TODO (user): Add any teardown logic common to all tests
	})

	Context("When creating or updating TalosControlPlane under Validating Webhook", func() {
		It("should return an error if the version is decreased", func() {
			oldObj.Spec.Version = "v1.5.0"
			obj.Spec.Version = "v1.4.0"
			_, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(HaveOccurred())
		})

		It("should not return an error if the version is increased", func() {
			oldObj.Spec.Version = "v1.5.0"
			obj.Spec.Version = "v1.6.0"
			_, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return an error if the kubeVersion is decreased", func() {
			oldObj.Spec.KubeVersion = "v1.28.0"
			obj.Spec.KubeVersion = "v1.27.0"
			_, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(HaveOccurred())
		})

		It("should not return an error if the kubeVersion is increased", func() {
			oldObj.Spec.KubeVersion = "v1.28.0"
			obj.Spec.KubeVersion = "v1.29.0"
			_, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return an error if the machineSpec.image is decreased", func() {
			image := "ghcr.io/siderolabs/talos:v1.5.0"
			oldObj.Spec.MetalSpec.MachineSpec.Image = &image
			image = "ghcr.io/siderolabs/talos:v1.4.0"
			obj.Spec.MetalSpec.MachineSpec.Image = &image
			_, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(HaveOccurred())
		})

		It("should not return an error if the machineSpec.image is increased", func() {
			image := "ghcr.io/siderolabs/talos:v1.5.0"
			oldObj.Spec.MetalSpec.MachineSpec.Image = &image
			image = "ghcr.io/siderolabs/talos:v1.6.0"
			obj.Spec.MetalSpec.MachineSpec.Image = &image
			_, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

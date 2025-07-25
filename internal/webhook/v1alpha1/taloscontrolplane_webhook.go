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
	"fmt"

	"github.com/alperencelik/talos-operator/pkg/utils"
	"github.com/hashicorp/go-version"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var taloscontrolplanelog = logf.Log.WithName("taloscontrolplane-resource")

// SetupTalosControlPlaneWebhookWithManager registers the webhook for TalosControlPlane in the manager.
func SetupTalosControlPlaneWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&talosv1alpha1.TalosControlPlane{}).
		WithValidator(&TalosControlPlaneCustomValidator{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.

// TalosControlPlaneCustomValidator struct is responsible for validating the TalosControlPlane resource
// when it is created, updated, or deleted.
//

// +kubebuilder:webhook:path=/validate-talos-alperen-cloud-v1alpha1-taloscontrolplane,mutating=false,failurePolicy=fail,sideEffects=None,groups=talos.alperen.cloud,resources=taloscontrolplanes,verbs=create;update,versions=v1alpha1,name=vtaloscontrolplane-v1alpha1.kb.io,admissionReviewVersions=v1

type TalosControlPlaneCustomValidator struct {
	//TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &TalosControlPlaneCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type TalosControlPlane.
func (v *TalosControlPlaneCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	taloscontrolplane, ok := obj.(*talosv1alpha1.TalosControlPlane)
	if !ok {
		return nil, fmt.Errorf("expected a TalosControlPlane object but got %T", obj)
	}
	taloscontrolplanelog.Info("Validation for TalosControlPlane upon creation", "name", taloscontrolplane.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type TalosControlPlane.
func (v *TalosControlPlaneCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	taloscontrolplane, ok := newObj.(*talosv1alpha1.TalosControlPlane)
	if !ok {
		return nil, fmt.Errorf("expected a TalosControlPlane object for the newObj but got %T", newObj)
	}
	taloscontrolplanelog.Info("Validation for TalosControlPlane upon update", "name", taloscontrolplane.GetName())

	old, ok := oldObj.(*talosv1alpha1.TalosControlPlane)
	if !ok {
		return nil, fmt.Errorf("expected a TalosControlPlane object for the oldObj but got %T", oldObj)
	}

	newVersion, err := version.NewVersion(taloscontrolplane.Spec.Version)
	if err != nil {
		return nil, fmt.Errorf("invalid new version: %w", err)
	}

	oldVersion, err := version.NewVersion(old.Spec.Version)
	if err != nil {
		return nil, fmt.Errorf("invalid old version: %w", err)
	}

	if newVersion.LessThan(oldVersion) {
		return nil, fmt.Errorf("version can not be decreased")
	}

	newKubeVersion, err := version.NewVersion(taloscontrolplane.Spec.KubeVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid new kubeVersion: %w", err)
	}

	oldKubeVersion, err := version.NewVersion(old.Spec.KubeVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid old kubeVersion: %w", err)
	}

	if newKubeVersion.LessThan(oldKubeVersion) {
		return nil, fmt.Errorf("kubeVersion can not be decreased")
	}

	newImageVersion, err := version.NewVersion(utils.GetVersionSuffix(*taloscontrolplane.Spec.MetalSpec.MachineSpec.Image))
	if err != nil {
		return nil, fmt.Errorf("invalid new image version: %w", err)
	}

	oldImageVersion, err := version.NewVersion(utils.GetVersionSuffix(*old.Spec.MetalSpec.MachineSpec.Image))
	if err != nil {
		return nil, fmt.Errorf("invalid old image version: %w", err)
	}

	if newImageVersion.LessThan(oldImageVersion) {
		return nil, fmt.Errorf("machineSpec.image can not be decreased")
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type TalosControlPlane.
func (v *TalosControlPlaneCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	taloscontrolplane, ok := obj.(*talosv1alpha1.TalosControlPlane)
	if !ok {
		return nil, fmt.Errorf("expected a TalosControlPlane object but got %T", obj)
	}
	taloscontrolplanelog.Info("Validation for TalosControlPlane upon deletion", "name", taloscontrolplane.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}

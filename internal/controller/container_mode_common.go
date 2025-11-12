package controller

import (
	"fmt"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func BuildServiceSpec(name string, i *int32) corev1.ServiceSpec {
	// If the index is nil, we assume it's a headless service for the StatefulSet.
	selectors := map[string]string{
		"app": name,
	}
	var svcType corev1.ServiceType
	if i != nil {
		selectors["statefulset.kubernetes.io/pod-name"] = fmt.Sprintf("%s-%d", name, *i)
		svcType = corev1.ServiceTypeClusterIP
	} else {
		// If i is nil, we are building a headless service for the StatefulSet and it should be LB type.
		svcType = corev1.ServiceTypeLoadBalancer
	}

	return corev1.ServiceSpec{
		Selector: selectors,
		Ports: []corev1.ServicePort{
			{
				Name:       "talos-api",
				Port:       50000,
				TargetPort: intstr.FromInt(50000),
			},
			{
				Name:       "k8s-api",
				Port:       6443,
				TargetPort: intstr.FromInt(6443),
			},
		},
		Type: svcType,
	}
}

func BuildUserDataEnvVar(configRef *corev1.ConfigMapKeySelector, name string, machineType string) []corev1.EnvVar {
	if configRef != nil {
		return []corev1.EnvVar{
			{
				Name: "USERDATA",
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: configRef.Name,
						},
						Key: configRef.Key,
					},
				},
			},
		}
	} else {
		var key string
		switch machineType {
		case TalosMachineTypeWorker:
			key = "worker.yaml"
		case TalosMachineTypeControlPlane:
			key = "controlplane.yaml"
		}
		return []corev1.EnvVar{
			{
				Name: "USERDATA",
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: fmt.Sprintf("%s-config", name),
						},
						Key: key,
					},
				},
			},
		}
	}
}

func BuildStsSpec(name string, replicas int32, version string, machineType string, extraEnvs []corev1.EnvVar, storageClassName *string) appsv1.StatefulSetSpec {
	return appsv1.StatefulSetSpec{
		ServiceName: name,
		Replicas:    &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": name,
			},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"app": name,
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "talos-control-plane",
						Image: fmt.Sprintf("%s:%s", TalosImage, version),
						Env: append(extraEnvs,
							corev1.EnvVar{
								Name:  TalosPlatformKey,
								Value: TalosModeContainer,
							},
						),
						Ports: []corev1.ContainerPort{
							{
								Name:          "talos-api",
								ContainerPort: 50000,
							},
							{
								Name:          "k8s-api",
								ContainerPort: 6443,
							},
						},
						SecurityContext: &corev1.SecurityContext{
							Privileged:             ptr.To(true),
							ReadOnlyRootFilesystem: ptr.To(true),
							SeccompProfile: &corev1.SeccompProfile{
								Type: corev1.SeccompProfileTypeUnconfined,
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "run",
								MountPath: "/run",
							},
							{
								Name:      "system",
								MountPath: "/system",
							},
							{
								Name:      "tmp",
								MountPath: "/tmp",
							},
							{
								Name:      "var",
								MountPath: "/var",
							},
							{
								Name:      "etc-cni",
								MountPath: "/etc/cni",
							},
							{
								Name:      "etc-kubernetes",
								MountPath: "/etc/kubernetes",
							},
							{
								Name:      "usr-libexec-kubernetes",
								MountPath: "/usr/libexec/kubernetes",
							},
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "run",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
					{
						Name: "system",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
					{
						Name: "tmp",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
			},
		},
		VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "system-state",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName: storageClassName,
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "var",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName: storageClassName,
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("20Gi"),
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "etc-cni",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName: storageClassName,
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "etc-kubernetes",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName: storageClassName,
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "usr-libexec-kubernetes",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName: storageClassName,
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
				},
			},
		},
	}
}
func BuildK8sUpgradeJobSpec(tcp *talosv1alpha1.TalosControlPlane, image, serviceAccount string) batchv1.JobSpec {
	return batchv1.JobSpec{
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				RestartPolicy: corev1.RestartPolicyOnFailure,
				Containers: []corev1.Container{
					{
						Name:  "upgrade",
						Image: image,
						Args: []string{
							"upgrade-k8s",
						},
						Env: []corev1.EnvVar{
							{
								Name:  "TARGET_VERSION",
								Value: tcp.Spec.KubeVersion,
							},
							{
								Name:  "TCP_NAME",
								Value: tcp.Name,
							},
							{
								Name:  "TCP_NAMESPACE",
								Value: tcp.Namespace,
							},
						},
					},
				},
				ServiceAccountName: serviceAccount,
			},
		},
		BackoffLimit: func(i int32) *int32 { return &i }(3),
	}
}

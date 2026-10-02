/*
Copyright 2026.

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
	"fmt"

	common_helper "github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	apiv1beta1 "github.com/openstack-k8s-operators/lightspeed-operator/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ReconcileOKPDeployment reconciles OKP and its MCP search server.
func ReconcileOKPDeployment(ctx context.Context, h *common_helper.Helper, instance *apiv1beta1.OpenStackLightspeed) error {
	tasks := []ReconcileTask{
		{Name: "OKPDeployment", Task: reconcileOKPDeployment},
		{Name: "OKPService", Task: reconcileOKPService},
		{Name: "OKPMCPDeployment", Task: reconcileOKPMCPDeployment},
		{Name: "OKPMCPService", Task: reconcileOKPMCPService},
	}
	return ReconcileTasksFailFast(ctx, h, instance, tasks)
}

func reconcileOKPDeployment(ctx context.Context, h *common_helper.Helper, instance *apiv1beta1.OpenStackLightspeed) error {
	logger := h.GetLogger()

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      OKPDeploymentName,
			Namespace: h.GetBeforeObject().GetNamespace(),
		},
	}

	result, err := controllerutil.CreateOrPatch(ctx, h.GetClient(), deployment, func() error {
		podTemplateSpec := buildOKPPodTemplateSpec(instance)

		replicas := int32(1)
		deployment.Spec.Replicas = &replicas
		deployment.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: generateOKPSelectorLabels(),
		}
		deployment.Spec.Template = podTemplateSpec

		return controllerutil.SetControllerReference(h.GetBeforeObject(), deployment, h.GetScheme())
	})

	if err != nil {
		return fmt.Errorf("%w: %w", ErrCreateOKPDeployment, err)
	}

	logger.Info("OKP Deployment reconciled", "name", deployment.Name, "result", result)
	return nil
}

func reconcileOKPService(ctx context.Context, h *common_helper.Helper, _ *apiv1beta1.OpenStackLightspeed) error {
	logger := h.GetLogger()

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      OKPServiceName,
			Namespace: h.GetBeforeObject().GetNamespace(),
		},
	}

	result, err := controllerutil.CreateOrPatch(ctx, h.GetClient(), svc, func() error {
		svc.Spec.Selector = generateOKPSelectorLabels()
		svc.Spec.Ports = []corev1.ServicePort{
			{
				Name:       "http",
				Port:       OKPServicePort,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromString("okp"),
			},
			{
				Name:       "solr",
				Port:       OKPSolrPort,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromString("solr"),
			},
		}
		svc.Spec.Type = corev1.ServiceTypeClusterIP

		return controllerutil.SetControllerReference(h.GetBeforeObject(), svc, h.GetScheme())
	})

	if err != nil {
		return fmt.Errorf("%w: %w", ErrCreateOKPService, err)
	}

	logger.Info("OKP Service reconciled", "name", svc.Name, "result", result)
	return nil
}

func buildOKPPodTemplateSpec(instance *apiv1beta1.OpenStackLightspeed) corev1.PodTemplateSpec {
	envVars := []corev1.EnvVar{{Name: "SOLR_JETTY_HOST", Value: "0.0.0.0"}}
	if instance.Spec.OKP != nil && instance.Spec.OKP.AccessKey != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name: "ACCESS_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: instance.Spec.OKP.AccessKey,
					},
					Key: OKPAccessKeySecretKey,
				},
			},
		})
	}

	resources := corev1.ResourceRequirements{}
	if instance.Spec.OKP != nil {
		resources = instance.Spec.OKP.Resources
	}

	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: generateOKPSelectorLabels(),
		},
		Spec: corev1.PodSpec{
			SecurityContext: &corev1.PodSecurityContext{
				RunAsNonRoot: toPtr(true),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			},
			AutomountServiceAccountToken: toPtr(false),
			Containers: []corev1.Container{
				{
					Name:  OKPContainerName,
					Image: instance.OKPContainerImage(),
					Ports: []corev1.ContainerPort{{Name: "okp", ContainerPort: OKPContainerPort}, {Name: "solr", ContainerPort: OKPSolrPort}},
					Env:   envVars,
					StartupProbe: &corev1.Probe{
						ProbeHandler:     corev1.ProbeHandler{TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromString("solr")}},
						PeriodSeconds:    10,
						FailureThreshold: 60,
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromString("solr")},
						},
						InitialDelaySeconds: 30,
						PeriodSeconds:       10,
					},
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/",
								Port: intstr.FromInt32(OKPContainerPort),
							},
						},
						InitialDelaySeconds: 60,
						PeriodSeconds:       20,
					},
					Resources:       resources,
					ImagePullPolicy: corev1.PullIfNotPresent,
					// NOTE: readOnlyRootFilesystem is intentionally not set for OKP.
					// The image mutates files under /etc/httpd/conf at startup.
					SecurityContext: &corev1.SecurityContext{
						RunAsNonRoot:             toPtr(true),
						AllowPrivilegeEscalation: toPtr(false),
						Capabilities: &corev1.Capabilities{
							Drop: []corev1.Capability{"ALL"},
						},
					},
				},
			},
		},
	}
}

func generateOKPMCPSelectorLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/component":  "okp-mcp",
		"app.kubernetes.io/managed-by": "openstack-lightspeed-operator",
		"app.kubernetes.io/name":       "openstack-lightspeed-okp-mcp",
		"app.kubernetes.io/part-of":    "openstack-lightspeed",
	}
}

func okpMCPResources(instance *apiv1beta1.OpenStackLightspeed) corev1.ResourceRequirements {
	resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("50m"), corev1.ResourceMemory: resource.MustParse("300Mi")},
		Limits:   corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("500Mi")},
	}
	if instance.Spec.OKP != nil && instance.Spec.OKP.MCP != nil {
		for name, quantity := range instance.Spec.OKP.MCP.Resources.Requests {
			resources.Requests[name] = quantity
		}
		for name, quantity := range instance.Spec.OKP.MCP.Resources.Limits {
			resources.Limits[name] = quantity
		}
	}
	return resources
}

func buildOKPMCPPodTemplateSpec(instance *apiv1beta1.OpenStackLightspeed, chunkFilterQuery string) corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{Labels: generateOKPMCPSelectorLabels()},
		Spec: corev1.PodSpec{
			AutomountServiceAccountToken: toPtr(false),
			Volumes:                      []corev1.Volume{{Name: "tmp", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}},
			SecurityContext: &corev1.PodSecurityContext{
				RunAsNonRoot:   toPtr(true),
				SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
			},
			Containers: []corev1.Container{{
				Name: "okp-mcp", Image: instance.OKPMCPContainerImage(), ImagePullPolicy: corev1.PullIfNotPresent,
				Ports:        []corev1.ContainerPort{{Name: "mcp", ContainerPort: OKPMCPPort}},
				VolumeMounts: []corev1.VolumeMount{{Name: "tmp", MountPath: "/tmp"}},
				Env: []corev1.EnvVar{
					{Name: "MCP_TRANSPORT", Value: "streamable-http"},
					{Name: "MCP_PORT", Value: "8000"},
					{Name: "MCP_SOLR_URL", Value: fmt.Sprintf("http://%s.%s.svc:%d", OKPServiceName, instance.Namespace, OKPSolrPort)},
					{Name: "MCP_OKP_SCOPE_FILTER", Value: chunkFilterQuery},
				},
				StartupProbe:   &corev1.Probe{ProbeHandler: corev1.ProbeHandler{TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromString("mcp")}}, PeriodSeconds: 10, FailureThreshold: 30},
				ReadinessProbe: &corev1.Probe{ProbeHandler: corev1.ProbeHandler{TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromString("mcp")}}, PeriodSeconds: 10},
				LivenessProbe:  &corev1.Probe{ProbeHandler: corev1.ProbeHandler{TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromString("mcp")}}, PeriodSeconds: 20},
				Resources:      okpMCPResources(instance),
				SecurityContext: &corev1.SecurityContext{
					RunAsNonRoot: toPtr(true), AllowPrivilegeEscalation: toPtr(false),
					ReadOnlyRootFilesystem: toPtr(true),
					Capabilities:           &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
				},
			}},
		},
	}
}

func reconcileOKPMCPDeployment(ctx context.Context, h *common_helper.Helper, instance *apiv1beta1.OpenStackLightspeed) error {
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: OKPMCPDeploymentName, Namespace: instance.Namespace}}
	_, err := controllerutil.CreateOrPatch(ctx, h.GetClient(), deployment, func() error {
		replicas := int32(1)
		deployment.Spec.Replicas = &replicas
		deployment.Spec.Selector = &metav1.LabelSelector{MatchLabels: generateOKPMCPSelectorLabels()}
		deployment.Spec.Template = buildOKPMCPPodTemplateSpec(instance, getOKPChunkFilterQuery(ctx, h, instance))
		return controllerutil.SetControllerReference(h.GetBeforeObject(), deployment, h.GetScheme())
	})
	return err
}

func reconcileOKPMCPService(ctx context.Context, h *common_helper.Helper, instance *apiv1beta1.OpenStackLightspeed) error {
	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: OKPMCPServiceName, Namespace: instance.Namespace}}
	_, err := controllerutil.CreateOrPatch(ctx, h.GetClient(), service, func() error {
		service.Spec.Selector = generateOKPMCPSelectorLabels()
		service.Spec.Type = corev1.ServiceTypeClusterIP
		service.Spec.Ports = []corev1.ServicePort{{Name: "mcp", Port: OKPMCPPort, Protocol: corev1.ProtocolTCP, TargetPort: intstr.FromString("mcp")}}
		return controllerutil.SetControllerReference(h.GetBeforeObject(), service, h.GetScheme())
	})
	return err
}

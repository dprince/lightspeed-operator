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
	"testing"

	apiv1beta1 "github.com/openstack-k8s-operators/lightspeed-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testRAGImage        = "example.com/rag:override"
	testOGXImage        = "example.com/ogx:override"
	testLightspeedImage = "example.com/lightspeed:override"
	testExporterImage   = "example.com/exporter:override"
	testPostgresImage   = "example.com/postgres:override"
	testOKPImage        = "example.com/okp:override"
	testOKPMCPImage     = "example.com/okp-mcp:override"
	testConsoleImage    = "example.com/console:override"
)

func setContainerImageTestDefaults(t *testing.T) {
	t.Helper()
	apiv1beta1.OpenStackLightspeedDefaultValues = apiv1beta1.OpenStackLightspeedDefaults{
		RAGImageURL:        "default/rag:1",
		LCoreImageURL:      "default/lcore:1",
		OGXImageURL:        "default/ogx:1",
		ExporterImageURL:   "default/exporter:1",
		PostgresImageURL:   "default/postgres:1",
		OKPImageURL:        "default/okp:1",
		OKPMCPImageURL:     "default/okp-mcp:1",
		ConsoleImageURL:    "default/console-pf6:1",
		ConsoleImagePF5URL: "default/console-pf5:1",
	}
}

func makeContainerImageTestInstance() *apiv1beta1.OpenStackLightspeed {
	feedbackDisabled := false
	return &apiv1beta1.OpenStackLightspeed{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: "test-ns",
		},
		Spec: apiv1beta1.OpenStackLightspeedSpec{
			OpenStackLightspeedCore: apiv1beta1.OpenStackLightspeedCore{
				LLMEndpoint:     "http://mock-llm:8000/v1",
				LLMEndpointType: "openai",
				ModelName:       "test-model",
				LLMCredentials:  "llm-secret",
				RAG:             &apiv1beta1.RAG{ContainerImage: testRAGImage},
				OGX:             &apiv1beta1.OGXSpec{ContainerImage: testOGXImage},
				LCore:           &apiv1beta1.LCoreSpec{ContainerImage: testLightspeedImage},
				DataverseExporter: &apiv1beta1.DataverseExporter{
					ContainerImage: testExporterImage,
					Feedback:       &apiv1beta1.DataverseExporterFeedback{Enabled: &feedbackDisabled},
				},
			},
			Console:  &apiv1beta1.ConsoleSpec{ContainerImage: testConsoleImage},
			Database: &apiv1beta1.DatabaseSpec{ContainerImage: testPostgresImage},
			OKP:      &apiv1beta1.OKPSpec{ContainerImage: testOKPImage, MCP: &apiv1beta1.OKPMCPSpec{ContainerImage: testOKPMCPImage}},
		},
	}
}

func TestBuildInitContainers_UsesContainerImageOverrides(t *testing.T) {
	setContainerImageTestDefaults(t)
	instance := makeContainerImageTestInstance()

	initContainers := buildInitContainers(instance, corev1.ResourceRequirements{})
	if len(initContainers) != 2 {
		t.Fatalf("expected 2 init containers, got %d", len(initContainers))
	}

	if got := initContainers[0].Image; got != testRAGImage {
		t.Errorf("vector-database-collect image = %q, want %q", got, testRAGImage)
	}
	if got := initContainers[1].Image; got != testLightspeedImage {
		t.Errorf("vector-database-config-build image = %q, want %q", got, testLightspeedImage)
	}
}

func TestBuildPostgresPodTemplateSpec_UsesContainerImageOverride(t *testing.T) {
	setContainerImageTestDefaults(t)
	instance := makeContainerImageTestInstance()

	podTemplate := buildPostgresPodTemplateSpec(instance)
	if len(podTemplate.Spec.Containers) != 1 {
		t.Fatalf("expected 1 postgres container, got %d", len(podTemplate.Spec.Containers))
	}
	if got := podTemplate.Spec.Containers[0].Image; got != testPostgresImage {
		t.Errorf("postgres image = %q, want %q", got, testPostgresImage)
	}
}

func TestBuildOKPPodTemplateSpec_UsesContainerImageOverride(t *testing.T) {
	setContainerImageTestDefaults(t)
	instance := makeContainerImageTestInstance()

	podTemplate := buildOKPPodTemplateSpec(instance)
	if len(podTemplate.Spec.Containers) != 1 {
		t.Fatalf("expected 1 okp container, got %d", len(podTemplate.Spec.Containers))
	}
	if got := podTemplate.Spec.Containers[0].Image; got != testOKPImage {
		t.Errorf("okp image = %q, want %q", got, testOKPImage)
	}
	container := podTemplate.Spec.Containers[0]
	if len(container.Ports) != 2 || container.Ports[1].ContainerPort != OKPSolrPort {
		t.Errorf("OKP does not expose the Solr port: %v", container.Ports)
	}
	if container.Env[0].Name != "SOLR_JETTY_HOST" || container.Env[0].Value != "0.0.0.0" {
		t.Errorf("OKP Solr is not bound to the pod network: %v", container.Env)
	}
}

func TestBuildOKPMCPPodTemplateSpec(t *testing.T) {
	setContainerImageTestDefaults(t)
	instance := makeContainerImageTestInstance()
	pod := buildOKPMCPPodTemplateSpec(instance, `product:"OpenStack Platform"`)
	container := pod.Spec.Containers[0]
	if container.Image != testOKPMCPImage {
		t.Fatalf("MCP image = %q, want %q", container.Image, testOKPMCPImage)
	}
	if container.Env[2].Value != "http://lightspeed-okp-server.test-ns.svc:8983" {
		t.Errorf("MCP Solr URL = %q", container.Env[2].Value)
	}
	if container.Env[3].Name != "MCP_OKP_SCOPE_FILTER" || container.Env[3].Value != `product:"OpenStack Platform"` {
		t.Errorf("MCP scope filter env var = %+v", container.Env[3])
	}
	if container.ReadinessProbe.TCPSocket.Port.StrVal != "mcp" {
		t.Errorf("MCP readiness probe does not check the MCP port")
	}
	if got := container.Resources.Requests.Memory().String(); got != "300Mi" {
		t.Errorf("MCP default memory request = %q", got)
	}
	instance.Spec.OKP.MCP.Resources.Requests = corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("400Mi")}
	resources := okpMCPResources(instance)
	if got := resources.Requests.Memory().String(); got != "400Mi" {
		t.Errorf("MCP memory override = %q", got)
	}
	if got := resources.Requests.Cpu().String(); got != "50m" {
		t.Errorf("MCP CPU default was lost: %q", got)
	}
}

func TestBuildConsoleDeploymentSpec_UsesContainerImageOverride(t *testing.T) {
	setContainerImageTestDefaults(t)
	instance := makeContainerImageTestInstance()

	spec := buildConsoleDeploymentSpec(testConsoleImage, instance)
	if len(spec.Template.Spec.InitContainers) != 1 {
		t.Fatalf("expected 1 init container, got %d", len(spec.Template.Spec.InitContainers))
	}
	if len(spec.Template.Spec.Containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(spec.Template.Spec.Containers))
	}

	if got := spec.Template.Spec.InitContainers[0].Image; got != testConsoleImage {
		t.Errorf("console init container image = %q, want %q", got, testConsoleImage)
	}
	if got := spec.Template.Spec.Containers[0].Image; got != testConsoleImage {
		t.Errorf("console container image = %q, want %q", got, testConsoleImage)
	}
}

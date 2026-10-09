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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testRAGImage        = "example.com/rag:override"
	testOGXImage        = "example.com/ogx:override"
	testLightspeedImage = "example.com/lightspeed:override"
	testExporterImage   = "example.com/exporter:override"
	testPostgresImage   = "example.com/postgres:override"
	testOKPImage        = "example.com/okp:override"
)

func setContainerImageTestDefaults(t *testing.T) {
	t.Helper()
	apiv1beta1.OpenStackLightspeedDefaultValues = apiv1beta1.OpenStackLightspeedDefaults{
		RAGImageURL:      "default/rag:1",
		LCoreImageURL:    "default/lcore:1",
		OGXImageURL:      "default/ogx:1",
		ExporterImageURL: "default/exporter:1",
		PostgresImageURL: "default/postgres:1",
		OKPImageURL:      "default/okp:1",
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
				DefaultModel: "test-model",
				Models: []apiv1beta1.OpenStackLightspeedModelSpec{{
					Name:            "test-model",
					LLMEndpoint:     "http://mock-llm:8000/v1",
					LLMEndpointType: "openai",
					LLMCredentials:  "llm-secret",
					ModelName:       "test-model",
				}},
				RAG:   &apiv1beta1.RAG{ContainerImage: testRAGImage},
				OGX:   &apiv1beta1.OGXSpec{ContainerImage: testOGXImage},
				LCore: &apiv1beta1.LCoreSpec{ContainerImage: testLightspeedImage},
				DataverseExporter: &apiv1beta1.DataverseExporter{
					ContainerImage: testExporterImage,
					Feedback:       &apiv1beta1.DataverseExporterFeedback{Enabled: &feedbackDisabled},
				},
			},
			Database: &apiv1beta1.DatabaseSpec{ContainerImage: testPostgresImage},
			OKP:      &apiv1beta1.OKPSpec{ContainerImage: testOKPImage},
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
}

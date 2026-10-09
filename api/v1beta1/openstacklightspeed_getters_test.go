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

package v1beta1

import (
	"testing"
)

func TestResolveContainerImage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		manifestImage  string
		defaultImage   string
		expectedResult string
	}{
		{
			name:           "manifest override",
			manifestImage:  "example.com/custom:1.0",
			defaultImage:   "example.com/default:1.0",
			expectedResult: "example.com/custom:1.0",
		},
		{
			name:           "empty manifest uses default",
			manifestImage:  "",
			defaultImage:   "example.com/default:1.0",
			expectedResult: "example.com/default:1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := resolveContainerImage(tt.manifestImage, tt.defaultImage); got != tt.expectedResult {
				t.Errorf("resolveContainerImage() = %q, want %q", got, tt.expectedResult)
			}
		})
	}
}

func TestOpenStackLightspeedContainerImages(t *testing.T) {
	OpenStackLightspeedDefaultValues = OpenStackLightspeedDefaults{
		RAGImageURL:      "default/rag:1",
		LCoreImageURL:    "default/lcore:1",
		OGXImageURL:      "default/ogx:1",
		ExporterImageURL: "default/exporter:1",
		PostgresImageURL: "default/postgres:1",
		OKPImageURL:      "default/okp:1",
	}

	instance := &OpenStackLightspeed{
		Spec: OpenStackLightspeedSpec{
			OpenStackLightspeedCore: OpenStackLightspeedCore{
				RAG:               &RAG{ContainerImage: "custom/rag:2"},
				OGX:               &OGXSpec{ContainerImage: "custom/ogx:2"},
				LCore:             &LCoreSpec{ContainerImage: "custom/lcore:2"},
				DataverseExporter: &DataverseExporter{ContainerImage: "custom/exporter:2"},
			},
			Database: &DatabaseSpec{ContainerImage: "custom/postgres:2"},
			OKP:      &OKPSpec{ContainerImage: "custom/okp:2"},
		},
	}

	if got := instance.RAGContainerImage(); got != "custom/rag:2" {
		t.Errorf("RAGContainerImage() = %q, want %q", got, "custom/rag:2")
	}
	if got := instance.OGXContainerImage(); got != "custom/ogx:2" {
		t.Errorf("OGXContainerImage() = %q, want %q", got, "custom/ogx:2")
	}
	if got := instance.LightspeedContainerImage(); got != "custom/lcore:2" {
		t.Errorf("LightspeedContainerImage() = %q, want %q", got, "custom/lcore:2")
	}
	if got := instance.ExporterContainerImage(); got != "custom/exporter:2" {
		t.Errorf("ExporterContainerImage() = %q, want %q", got, "custom/exporter:2")
	}
	if got := instance.PostgresContainerImage(); got != "custom/postgres:2" {
		t.Errorf("PostgresContainerImage() = %q, want %q", got, "custom/postgres:2")
	}
	if got := instance.OKPContainerImage(); got != "custom/okp:2" {
		t.Errorf("OKPContainerImage() = %q, want %q", got, "custom/okp:2")
	}
}

func TestOpenStackLightspeedContainerImagesUseDefaults(t *testing.T) {
	OpenStackLightspeedDefaultValues = OpenStackLightspeedDefaults{
		RAGImageURL:      "default/rag:1",
		LCoreImageURL:    "default/lcore:1",
		OGXImageURL:      "default/ogx:1",
		ExporterImageURL: "default/exporter:1",
		PostgresImageURL: "default/postgres:1",
		OKPImageURL:      "default/okp:1",
	}

	instance := &OpenStackLightspeed{
		Spec: OpenStackLightspeedSpec{},
	}

	if got := instance.RAGContainerImage(); got != "default/rag:1" {
		t.Errorf("RAGContainerImage() = %q, want %q", got, "default/rag:1")
	}
	if got := instance.OGXContainerImage(); got != "default/ogx:1" {
		t.Errorf("OGXContainerImage() = %q, want %q", got, "default/ogx:1")
	}
	if got := instance.LightspeedContainerImage(); got != "default/lcore:1" {
		t.Errorf("LightspeedContainerImage() = %q, want %q", got, "default/lcore:1")
	}
	if got := instance.ExporterContainerImage(); got != "default/exporter:1" {
		t.Errorf("ExporterContainerImage() = %q, want %q", got, "default/exporter:1")
	}
	if got := instance.PostgresContainerImage(); got != "default/postgres:1" {
		t.Errorf("PostgresContainerImage() = %q, want %q", got, "default/postgres:1")
	}
	if got := instance.OKPContainerImage(); got != "default/okp:1" {
		t.Errorf("OKPContainerImage() = %q, want %q", got, "default/okp:1")
	}
}

func TestSetupDefaults_OGXImageURLFromEnv(t *testing.T) {
	t.Setenv("RELATED_IMAGE_OGX_IMAGE_URL_DEFAULT", "env/ogx:1")
	t.Setenv("RELATED_IMAGE_LCORE_IMAGE_URL_DEFAULT", "env/lcore:1")
	t.Setenv("RELATED_IMAGE_OPENSTACK_LIGHTSPEED_IMAGE_URL_DEFAULT", "env/rag:1")
	t.Setenv("RELATED_IMAGE_EXPORTER_IMAGE_URL_DEFAULT", "env/exporter:1")
	t.Setenv("RELATED_IMAGE_POSTGRES_IMAGE_URL_DEFAULT", "env/postgres:1")
	t.Setenv("RELATED_IMAGE_OKP_IMAGE_URL_DEFAULT", "env/okp:1")
	t.Setenv("RELATED_IMAGE_MCP_SERVER_IMAGE_URL_DEFAULT", "env/mcp:1")

	SetupDefaults()

	if got := OpenStackLightspeedDefaultValues.OGXImageURL; got != "env/ogx:1" {
		t.Errorf("OGXImageURL = %q, want %q", got, "env/ogx:1")
	}
	if got := OpenStackLightspeedDefaultValues.LCoreImageURL; got != "env/lcore:1" {
		t.Errorf("LCoreImageURL = %q, want %q", got, "env/lcore:1")
	}
}

func TestOpenStackLightspeedContainerImages_OGXAndLightspeedIndependent(t *testing.T) {
	OpenStackLightspeedDefaultValues = OpenStackLightspeedDefaults{
		LCoreImageURL: "default/lcore:1",
		OGXImageURL:   "default/ogx:1",
	}

	instance := &OpenStackLightspeed{
		Spec: OpenStackLightspeedSpec{
			OpenStackLightspeedCore: OpenStackLightspeedCore{
				OGX:   &OGXSpec{ContainerImage: "custom/ogx:2"},
				LCore: &LCoreSpec{ContainerImage: "custom/lcore:2"},
			},
		},
	}

	if got := instance.OGXContainerImage(); got != "custom/ogx:2" {
		t.Errorf("OGXContainerImage() = %q, want %q", got, "custom/ogx:2")
	}
	if got := instance.LightspeedContainerImage(); got != "custom/lcore:2" {
		t.Errorf("LightspeedContainerImage() = %q, want %q", got, "custom/lcore:2")
	}
}

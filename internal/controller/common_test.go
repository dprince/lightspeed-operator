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
	"fmt"
	"strings"
	"testing"
	"time"

	apiv1beta1 "github.com/openstack-k8s-operators/lightspeed-operator/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestOKPChunkFilterQueryFmtExcludesOpenShiftVirtualization guards against OKP RAG
// answers being grounded in OpenShift Virtualization docs when the query is about
// OpenStack. OpenShift Virtualization docs share the same "openshift_container_platform"
// product as docs we want to keep (e.g. Migration Toolkit for Containers), so they can
// only be distinguished by their parent_id path (".../html-single/virtualization/index").
func TestOKPChunkFilterQueryFmtExcludesOpenShiftVirtualization(t *testing.T) {
	query := fmt.Sprintf(OKPChunkFilterQueryFmt, "18.0", "4.21")

	if !strings.Contains(query, "product:*openstack* AND product_version:18.0") {
		t.Errorf("expected OpenStack clause with version 18.0, got: %s", query)
	}
	if !strings.Contains(query, "product:*openshift*") {
		t.Errorf("expected OpenShift clause, got: %s", query)
	}
	if !strings.Contains(query, "product_version:4.21") {
		t.Errorf("expected OpenShift clause with version 4.21, got: %s", query)
	}
	if !strings.Contains(query, "-parent_id:*html-single/virtualization/*") {
		t.Errorf("expected OpenShift Virtualization docs to be excluded via parent_id, got: %s", query)
	}
}

func TestGenerateRandomStringLength(t *testing.T) {
	t.Run("Below minimum length returns error", func(t *testing.T) {
		_, err := generateRandomString(15)
		if err == nil {
			t.Error("generateRandomString(15) expected error, got nil")
		}
	})

	tests := []struct {
		name   string
		length int
	}{
		{name: "Min length 16", length: 16},
		{name: "Even length 32", length: 32},
		{name: "Odd length 33", length: 33},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := generateRandomString(tt.length)
			if err != nil {
				t.Errorf("generateRandomString(%d) unexpected error: %v", tt.length, err)
			}
			if len(result) != tt.length {
				t.Errorf("generateRandomString(%d) returned length %d, want %d", tt.length, len(result), tt.length)
			}
		})
	}
}

func TestGenerateRandomStringCharacters(t *testing.T) {
	result, err := generateRandomString(32)
	if err != nil {
		t.Fatalf("generateRandomString(32) unexpected error: %v", err)
	}

	var hasLower, hasUpper, hasDigit bool
	for i, c := range result {
		switch {
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= '0' && c <= '9':
			hasDigit = true
		default:
			t.Errorf("generateRandomString(32) character at index %d is %q, not alphanumeric", i, c)
		}
	}
	if !hasLower {
		t.Error("generateRandomString(32) result has no lowercase letter")
	}
	if !hasUpper {
		t.Error("generateRandomString(32) result has no uppercase letter")
	}
	if !hasDigit {
		t.Error("generateRandomString(32) result has no digit")
	}
}

func TestGenerateRandomStringUniqueness(t *testing.T) {
	const length = 16
	a, err := generateRandomString(length)
	if err != nil {
		t.Fatalf("first call unexpected error: %v", err)
	}
	b, err := generateRandomString(length)
	if err != nil {
		t.Fatalf("second call unexpected error: %v", err)
	}
	if a == b {
		t.Errorf("generateRandomString(%d) returned identical values across two calls: %q", length, a)
	}
}

func TestGetRawClientReturnsCachedClient(t *testing.T) {
	cached := fake.NewClientBuilder().Build()

	clusterClientMu.Lock()
	prev := clusterClient
	clusterClient = cached
	clusterClientMu.Unlock()
	t.Cleanup(func() {
		clusterClientMu.Lock()
		clusterClient = prev
		clusterClientMu.Unlock()
	})

	got, err := getRawClient(nil)
	if err != nil {
		t.Fatalf("getRawClient() unexpected error: %v", err)
	}
	if got != cached {
		t.Fatal("getRawClient() did not return the cached cluster client")
	}

	got2, err := getRawClient(nil)
	if err != nil {
		t.Fatalf("getRawClient() second call unexpected error: %v", err)
	}
	if got2 != got {
		t.Fatal("getRawClient() returned a different client on second call")
	}
}

func TestGetResourcePollInterval(t *testing.T) {
	makeInstance := func(devJSON string) *apiv1beta1.OpenStackLightspeed {
		instance := &apiv1beta1.OpenStackLightspeed{}
		if devJSON != "" {
			instance.Spec.Dev = runtime.RawExtension{Raw: []byte(devJSON)}
		}
		return instance
	}

	tests := []struct {
		name    string
		devJSON string
		want    time.Duration
	}{
		{name: "unset defaults to 60s", devJSON: "", want: ResourceCreationTimeout},
		{name: "empty object defaults to 60s", devJSON: `{}`, want: ResourceCreationTimeout},
		{name: "explicit 30s", devJSON: `{"resourcePollInterval":30}`, want: 30 * time.Second},
		{name: "explicit 120s", devJSON: `{"resourcePollInterval":120}`, want: 120 * time.Second},
		{name: "zero defaults to 60s", devJSON: `{"resourcePollInterval":0}`, want: ResourceCreationTimeout},
		{name: "negative defaults to 60s", devJSON: `{"resourcePollInterval":-1}`, want: ResourceCreationTimeout},
		{name: "malformed JSON defaults to 60s", devJSON: `{`, want: ResourceCreationTimeout},
		// 1e10 * time.Second overflows int64 nanoseconds to a negative duration.
		{name: "overflow defaults to 60s", devJSON: `{"resourcePollInterval":10000000000}`, want: ResourceCreationTimeout},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getResourcePollInterval(makeInstance(tt.devJSON))
			if got != tt.want {
				t.Errorf("getResourcePollInterval() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Ensure resourcePollInterval unmarshals from DevSpec via ParseDevConfig.
func TestParseDevConfigResourcePollInterval(t *testing.T) {
	instance := &apiv1beta1.OpenStackLightspeed{
		Spec: apiv1beta1.OpenStackLightspeedSpec{
			Dev: runtime.RawExtension{Raw: []byte(`{"resourcePollInterval":45}`)},
		},
	}
	devConfig, err := instance.ParseDevConfig()
	if err != nil {
		t.Fatalf("ParseDevConfig() unexpected error: %v", err)
	}
	if devConfig.ResourcePollInterval != 45 {
		t.Errorf("ResourcePollInterval = %d, want 45", devConfig.ResourcePollInterval)
	}
}

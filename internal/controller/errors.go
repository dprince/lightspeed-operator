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

import "errors"

var (
	// ErrCreateAPIConfigmap is returned when OpenStack Lightspeed ConfigMap cannot be created.
	ErrCreateAPIConfigmap = errors.New("failed to create OpenStack Lightspeed configmap")

	// ErrCreateAPIDeployment is returned when OpenStack Lightspeed Deployment cannot be created.
	ErrCreateAPIDeployment = errors.New("failed to create OpenStack Lightspeed deployment")

	// ErrCreateAPIService is returned when OpenStack Lightspeed Service cannot be created.
	ErrCreateAPIService = errors.New("failed to create OpenStack Lightspeed service")

	// ErrCreateAPIServiceAccount is returned when OpenStack Lightspeed ServiceAccount cannot be created.
	ErrCreateAPIServiceAccount = errors.New("failed to create OpenStack Lightspeed service account")

	// ErrCreateAppServerNetworkPolicy is returned when the AppServer NetworkPolicy cannot be created.
	ErrCreateAppServerNetworkPolicy = errors.New("failed to create AppServer network policy")

	// ErrCreateSARClusterRole is returned when the SAR ClusterRole cannot be created.
	ErrCreateSARClusterRole = errors.New("failed to create SAR cluster role")

	// ErrCreateSARClusterRoleBinding is returned when the SAR ClusterRoleBinding cannot be created.
	ErrCreateSARClusterRoleBinding = errors.New("failed to create SAR cluster role binding")

	// ErrDeleteSARClusterRole is returned when the SAR ClusterRole cannot be deleted.
	ErrDeleteSARClusterRole = errors.New("failed to delete SAR cluster role")

	// ErrDeleteSARClusterRoleBinding is returned when the SAR ClusterRoleBinding cannot be deleted.
	ErrDeleteSARClusterRoleBinding = errors.New("failed to delete SAR cluster role binding")

	// ErrGenerateAPIConfigmap is returned when the OpenStack Lightspeed ConfigMap cannot be generated.
	ErrGenerateAPIConfigmap = errors.New("failed to generate OpenStack Lightspeed configmap")

	// ErrGetTLSSecret is returned when the TLS Secret cannot be retrieved.
	ErrGetTLSSecret = errors.New("failed to get TLS secret")

	// ErrCreateOGXConfigMap is returned when the OGX ConfigMap cannot be created.
	ErrCreateOGXConfigMap = errors.New("failed to create OGX configmap")

	// ErrGenerateOGXConfigMap is returned when the OGX ConfigMap cannot be generated.
	ErrGenerateOGXConfigMap = errors.New("failed to generate OGX configmap")

	// ErrCreateExporterConfigMap is returned when the exporter ConfigMap cannot be created.
	ErrCreateExporterConfigMap = errors.New("failed to create exporter configmap")

	// ErrReadSystemCABundle is returned when the system CA bundle cannot be read.
	ErrReadSystemCABundle = errors.New("failed to read system CA bundle")

	// ErrParseSystemCABundle is returned when the system CA bundle cannot be parsed.
	ErrParseSystemCABundle = errors.New("failed to parse system CA bundle")

	// ErrParseUserCA is returned when the user CA certificate cannot be parsed.
	ErrParseUserCA = errors.New("failed to parse user CA certificate")

	// ErrCreateCABundle is returned when the CA bundle ConfigMap cannot be created.
	ErrCreateCABundle = errors.New("failed to create CA bundle configmap")

	// ErrGetCAConfigMap is returned when the CA ConfigMap cannot be retrieved.
	ErrGetCAConfigMap = errors.New("failed to get CA configmap")

	// ErrCreateOKPDeployment is returned when the OKP Deployment cannot be created.
	ErrCreateOKPDeployment = errors.New("failed to create OKP deployment")

	// ErrCreateOKPService is returned when the OKP Service cannot be created.
	ErrCreateOKPService = errors.New("failed to create OKP service")

	// ErrCreatePostgresDeployment is returned when the Postgres Deployment cannot be created.
	ErrCreatePostgresDeployment = errors.New("failed to create Postgres deployment")

	// ErrCreatePostgresService is returned when the Postgres Service cannot be created.
	ErrCreatePostgresService = errors.New("failed to create Postgres service")

	// ErrGeneratePostgresSecret is returned when the Postgres Secret cannot be generated.
	ErrGeneratePostgresSecret = errors.New("failed to generate Postgres secret")

	// ErrCreatePostgresSecret is returned when the Postgres Secret cannot be created.
	ErrCreatePostgresSecret = errors.New("failed to create Postgres secret")

	// ErrGetPostgresSecret is returned when the Postgres Secret cannot be retrieved.
	ErrGetPostgresSecret = errors.New("failed to get Postgres secret")

	// ErrCreatePostgresBootstrapSecret is returned when the Postgres bootstrap Secret cannot be created.
	ErrCreatePostgresBootstrapSecret = errors.New("failed to create Postgres bootstrap secret")

	// ErrCreatePostgresConfigMap is returned when the Postgres ConfigMap cannot be created.
	ErrCreatePostgresConfigMap = errors.New("failed to create Postgres configmap")

	// ErrGetPostgresConfigMap is returned when the Postgres ConfigMap cannot be retrieved.
	ErrGetPostgresConfigMap = errors.New("failed to get Postgres configmap")

	// ErrCreatePostgresNetworkPolicy is returned when the Postgres NetworkPolicy cannot be created.
	ErrCreatePostgresNetworkPolicy = errors.New("failed to create Postgres network policy")

	// ErrCreatePostgresPVC is returned when the Postgres PVC cannot be created.
	ErrCreatePostgresPVC = errors.New("failed to create Postgres PVC")

	// ErrPatchPostgresPVC is returned when the Postgres PVC cannot be patched.
	ErrPatchPostgresPVC = errors.New("failed to patch Postgres PVC")

	// ErrGetPostgresPVC is returned when the Postgres PVC cannot be retrieved.
	ErrGetPostgresPVC = errors.New("failed to get Postgres PVC")

	// ErrPostgresPVCSizeShrink is returned when an attempt is made to shrink an existing Postgres PVC.
	ErrPostgresPVCSizeShrink = errors.New("cannot shrink existing Postgres PVC")
)

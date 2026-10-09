# Development

This page is for contributors and anyone testing changes locally — not
needed if you're just installing and using the operator.

## Local cluster with CRC

For local development/testing only (not for trying the assistant for real
— CRC is resource-constrained). Deploy a CRC cluster before
[installing the operator](install_guide.md#installing-the-operator):

```bash
git clone https://github.com/openstack-k8s-operators/install_yamls.git
cd install_yamls/devsetup
make download_tools

CRC_VERSION=2.51.0 PULL_SECRET=~/work/pull-secret CRC_MONITORING_ENABLED=true CPUS=12 MEMORY=25600 DISK=100 make crc
make crc_attach_default_interface
eval $(crc oc-env)
cd ../..
```

`PULL_SECRET` is the same pull secret described in the
[installation guide](install_guide.md#access-to-registry-images).

## Architecture

```mermaid
graph TB
    User[System Administrator] --> Goose[Goose CLI]

    subgraph ns["openstack-lightspeed namespace"]
        CR[OpenStackLightspeed CR] --> Operator[lightspeed-operator]
        Operator --> DB[(PostgreSQL)]
        Operator --> OKP[OKP]
        Operator --> Pod

        subgraph Pod["lightspeed-stack pod"]
            API[lightspeed-service-api] --> OGX
            OGX -.-> MCP[MCP tools sidecar]
        end
    end

    Goose --> API
    OGX --> OKP
    OGX --> LLM[Your LLM endpoint]
    MCP -.->|read-only, optional| OSP[Your OpenStack / OpenShift APIs]
```

**OKP is deployed on every install, not opt-in.** It's the default RAG
source; the bundled community documentation is available too, but only if
you explicitly opt in. See [Configuration](configuration.md) for details.

## RHOKP container image

When it comes to the [Red Hat Offline Knowledge Portal (RHOKP or sometimes OKP) container image][rhokp-image],
and OpenStack Lightspeed, the following information might be relevant
for development:

- The [RHOKP container image][rhokp-image] has a time-based release with
  a weekly cadence. For more detailed information, consult the
  [RHOKP documentation][rhokp-docs].

- Only **the latest version** of the [RHOSO documentation][rhoso-docs] available at
  [docs.redhat.com][redhat-docs] during the [RHOKP container image][rhokp-image]
  build time is present in the container image.

- Only **the latest 7 versions** of the [OCP documentation][ocp-docs] available at
  [docs.redhat.com][redhat-docs] during the RHOKP container image build time
  are present in the container image.

## Pod security defaults

Operator-managed workloads are hardened by default with explicit
`securityContext` settings:

- `runAsNonRoot: true`
- `allowPrivilegeEscalation: false`
- `capabilities.drop: ["ALL"]`
- Pod-level `seccompProfile.type: RuntimeDefault` where supported

`readOnlyRootFilesystem: true` is enabled for most containers where
validated. Current intentional exceptions are:

- **OKP** container (runtime writes to image-managed paths)
- **MCP** sidecar (optional dev feature; may need mutable runtime paths)

If you change container startup behavior or image layout, re-validate
writable paths before enabling/disabling `readOnlyRootFilesystem`.

[rhokp-image]: https://catalog.redhat.com/en/software/containers/offline-knowledge-portal/rhokp-rhel9/680143c03b895f32bbd104c2
[rhokp-docs]: https://docs.redhat.com/en/documentation/red_hat_offline_knowledge_portal
[redhat-docs]: https://docs.redhat.com/en
[ocp-docs]: https://docs.redhat.com/en/documentation/openshift_container_platform
[rhoso-docs]: https://docs.redhat.com/en/documentation/red_hat_openstack_services_on_openshift

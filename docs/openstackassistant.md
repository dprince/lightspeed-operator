# Testing OpenStackAssistant with OpenStack Lightspeed

This guide deploys OpenStack Lightspeed and an OpenStack Assistant instance so
you can run Goose inside the cluster and try its OpenStack and OpenShift MCP
tools.

OpenStack Lightspeed provides the inference API Goose uses. OpenStackAssistant
provides the Goose runtime and its tool configuration; it is reconciled by the
OpenStack Operator. The MCP server used by Goose in this guide comes from
OpenStackClient. OpenStack Lightspeed has a separate optional MCP sidecar, which
this guide does not enable.

> [!IMPORTANT]
> Before starting, you need a working RHOSO deployment. The MCP tools in this
> guide expect resources from the deployed OpenStack control plane and are not
> available without RHOSO.

## 1. Deploy the OpenStack Lightspeed operator

From the lightspeed-operator repository, deploy the operator using the adjacent
`install_yamls` checkout:

```bash
(cd ../install_yamls && make openstack_lightspeed)
```

## 2. Create the LLM credentials and OpenStackLightspeed instance

Create the API token Secret in the `openstack-lightspeed` namespace. Replace
`REDACTED` locally with the API token for your LLM provider before applying;
do not commit the real token or add it to this document.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: openstack-lightspeed-apitoken
  namespace: openstack-lightspeed
type: Opaque
stringData:
  apitoken: REDACTED
```

Save this as `openstack-lightspeed-apitoken.yaml`.

Copy the maintained [OpenStackLightspeed sample](https://github.com/openstack-k8s-operators/lightspeed-operator/blob/main/config/samples/lightspeed_v1beta1_openstacklightspeed.yaml)
to `openstacklightspeed.yaml`. Set `metadata.name` to `openstack-lightspeed`
and `metadata.namespace` to `openstack-lightspeed`. Set `spec.defaultModel` to
the name of a model in `spec.models`, then set that model's `llmEndpoint`,
`llmEndpointType`, `llmCredentials`, and `modelName` for your provider. Use the
provider's API URL, including `/v1` when required. The endpoint must be
reachable from the `openstack-lightspeed` namespace. See
[Configuration](configuration.md#models-models) for field details and supported
providers.

If the endpoint uses a private or self-signed certificate that the cluster
does not trust, create the `llm-endpoint-ca` ConfigMap in
`openstack-lightspeed` with its CA bundle in PEM format and set
`spec.tlsCACertBundle` to `llm-endpoint-ca`. Otherwise, remove the sample's
`tlsCACertBundle` field.

Prepare `llm-endpoint-ca.yaml` locally from your provider's CA bundle if one is
needed, and keep credentials and certificate material out of version control.
Apply the Secret and custom resource from the lightspeed-operator repository
root. Apply the CA ConfigMap only when needed:

```bash
oc apply -f openstack-lightspeed-apitoken.yaml
oc apply -f llm-endpoint-ca.yaml # only for a private or self-signed endpoint certificate
oc apply -f openstacklightspeed.yaml
```

## 3. Wait for OpenStack Lightspeed to become ready

Wait for the custom resource to report `Ready`, then check its pods and
conditions for errors before proceeding:

```bash
oc wait -n openstack-lightspeed \
  openstacklightspeed/openstack-lightspeed \
  --for=condition=Ready --timeout=20m
oc get -n openstack-lightspeed deployments,pods
oc describe -n openstack-lightspeed \
  openstacklightspeed/openstack-lightspeed
```

Resolve any failing conditions, pod errors, or image pull issues before
creating the assistant.

## 4. Deploy OpenStackAssistant

The `openstack_assistant` target creates the `OpenStackAssistant` custom
resource and its supporting configuration. It expects the RHOSO
`OpenStackClient` resource named `openstackclient` in the `openstack`
namespace. The target checks that the OpenStackClient MCP endpoint is enabled
and enables it through the owning `OpenStackControlPlane` if needed.

```bash
(cd ../install_yamls && make openstack_assistant)
```

Confirm the assistant and its pod are ready before entering the pod:

```bash
oc get -n openstack openstackassistant/openstack-assistant
oc get -n openstack pod/openstack-assistant
```

## 5. Start Goose in the assistant pod

Enter the pod:

```bash
oc rsh -n openstack openstack-assistant
```

Then launch Goose in the pod:

```bash
goose
```

You can use interactive chat directly or run a pre-configured recipe with:

```bash
/cluster-health
```

The assistant can use the read-only OpenStack MCP server running alongside
OpenStackClient in its `mcp-server` sidecar. A Kubernetes NetworkPolicy protects
that MCP connection. The Goose developer extension also provides `oc` tools;
those run under a locked-down service account created by openstack-operator and
are restricted to read-only commands.

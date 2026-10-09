# Available Features

Goose CLI is the primary supported interface for RHOSO 19 Beta.
Available features include documentation questions, quotas, and data collection.

## Asking questions

Use Goose CLI once `OpenStackLightspeed` is `Ready` to ask questions such as:

- "How can I spin up a VM using the OpenStack CLI?"
- "Why would a Nova compute service show as down?"

Answers are grounded via RAG, with references you can verify. By default,
grounding comes from the [Offline knowledge portal](configuration.md#offline-knowledge-portal) (always deployed, no
credentials needed to browse — see [Configuration](configuration.md) for the free vs.
keyed tiers). The bundled community documentation is also available, but
only if you set `dev.okpRagOnly: false`.

## Multi-model request routing

When you configure multiple entries in `spec.models[]`, each entry defines:

- a **model alias** (`spec.models[].name`)
- a generated OGX **provider ID**: `provider-<alias>`
- the upstream provider model name (`spec.models[].modelName`)

In request payloads for `/query` and `/streaming_query`, use these fields to
select a non-default model:

- `provider`: the generated provider ID (`provider-<alias>`)
- `model`: the model alias (`<alias>`)

Example (explicit model selection):

```json
{
  "query": "How can I check Nova services?",
  "provider": "provider-another-model",
  "model": "another-model"
}
```

If `provider` and `model` are omitted, Lightspeed uses
`spec.defaultModel` (and its corresponding provider
`provider-<defaultModel>`).

## Quota enforcement (optional)

OpenStack Lightspeed can enforce token quotas per user and across the whole
cluster using lightspeed-stack's built-in quota system. The operator manages
the quota storage automatically, so no additional setup is needed.

Quota enforcement is opt-in: it is disabled until you configure at least one
limiter. You can combine per-user and cluster-wide limiters; requests must
satisfy each configured limiter. See [Quota enforcement](configuration.md#quota-enforcement) for the
configuration and an example.

## Feedback and transcripts

- `dataverseExporter.feedback.enabled` (default `true`) — thumbs-up/down on
  responses.
- `dataverseExporter.transcripts.enabled` (default `false`) — full
  conversation transcripts.

Both configured on the CR ([Configuration](configuration.md)). Used to improve answer
quality — disable either if that doesn't fit your data policy.

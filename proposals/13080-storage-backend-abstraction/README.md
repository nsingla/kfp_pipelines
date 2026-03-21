# KEP-13080: Extensible Multi-Backend Object Storage (Azure/PVC) with Unified Storage Layer

## Summary

Kubeflow Pipelines needs to support additional storage backends (notably Azure Blob and PVC/filesystem-backed storage)
without introducing repeated refactors across API server, driver, and launcher.

Today, storage-opening logic exists in two separate paths:

1. API server object store initialization in `backend/src/apiserver/client_manager/client_manager.go` (S3-compatible centric).
2. Driver/launcher artifact store opening in `backend/src/v2/objectstore/object_store.go` (S3/MinIO/GCS aware).

This proposal defines an extensible provider model where adding a backend (for example `azblob://` or `file://`) is
provider work, not cross-stack rewiring. To make that practical and safe, both existing call paths are unified behind a
shared provider abstraction in `backend/src/common/objectstore`.

Phase 1 keeps existing behavior intact for `s3://`, `minio://`, and `gs://` while laying the extension foundation.

The key outcomes are:

- no intended runtime regression for current providers;
- unchanged secret handling semantics;
- a clear path to add Azure Blob (`azblob://`) and PVC/filesystem (`file://`) backends without broad refactors.

## Motivation

Primary motivation: Kuberbetes is cloud agnostic, so should KFP especially when dealing with object storage and in order to do that, we need to support more storage backends (Azure Blob and PVC/filesystem) in a maintainable way.

Current duplication in provider/credential logic across API server and runtime makes backend expansion expensive and
risky:

- same provider implemented in two places with slightly different defaults and fallbacks;
- secret/auth code path divergence risk;
- higher change surface when adding new providers.

Unifying the opening layer is therefore not the end goal by itself; it is the enabling mechanism for safe backend
expansion while preserving current `*blob.Bucket` behavior and upload/download code paths.

## External Validation (Feasibility Check)

This proposal is validated against upstream behavior:

1. **Azure Blob fit**: Go CDK blob supports Azure via `azblob://` through `gocloud.dev/blob/azureblob` with `blob.OpenBucket`.
2. **Provider portability fit**: Go CDK URL mux model is provider-pluggable by scheme.
3. **PVC fit with caveat**: Go CDK supports filesystem buckets via `gocloud.dev/blob/fileblob` (`file://`). This fits PVC
   conceptually, but operational constraints (mount visibility and access modes) are Kubernetes deployment concerns, not
   generic object-store concerns.
4. **PVC constraints**: Kubernetes access modes (`RWO`, `RWX`, `RWOP`) and topology must be modeled explicitly before enabling.

## Goals

1. Enable future backend support (Azure Blob and PVC/filesystem) without requiring another cross-stack refactor.
2. Introduce a shared provider abstraction used by both API server and v2 runtime as the enabling architecture.
3. Preserve behavior for `s3://`, `minio://`, and `gs://`.
4. Preserve existing secret/credential semantics and fallback order.
5. Keep rollout incremental and low risk (no broad refactor of artifact upload/download logic).
6. Make provider onboarding additive via registry-based provider implementations.

## Non-Goals

1. Enabling Azure Blob runtime support in Phase 1.
2. Enabling PVC/filesystem runtime support in Phase 1.
3. Redesigning artifact read/write/list algorithms.
4. Changing user-facing artifact URI formats in Phase 1.
5. Changing frontend behavior in Phase 1.

## Requirements and Constraints

### Hard Requirements

1. No regression for existing paths and current install defaults.
2. Secrets handled with the same semantics as current implementation.
3. Keep solution simple and reuse existing code.
4. Make extension to new providers straightforward.

### Technical Constraints

1. API server still consumes `OBJECTSTORECONFIG_*` env vars.
2. Runtime still consumes `kfp-launcher` provider/session config and secret refs.
3. `BlobObjectStore` API in API server must remain stable.
4. Existing runtime artifact path semantics (`Prefix`, `KeyFromURI`, path traversal protections) must be unchanged.

## Current State (Before Change)

### API Server Path

- `initBlobObjectStore()` reads `OBJECTSTORECONFIG_*` env vars.
- `ensureBucketExists()` is best-effort.
- `openBucketWithRetry()` creates an S3 client and opens bucket.
- Object operations use `storage.BlobObjectStore`.

### Runtime Path (Driver/Launcher)

- `OpenBucket()` in `backend/src/v2/objectstore/object_store.go`:
  - switches provider (`minio/s3/gs`);
  - loads secrets for S3/GCS as needed;
  - opens prefixed bucket.
- Upload/download operations are implemented on top of `*blob.Bucket` and must remain intact.

## Proposal

Introduce `backend/src/common/objectstore` with:

- `Provider` interface and `Registry`.
- Shared request/config/session types.
- Capability metadata.
- Provider implementations:
  - `S3Provider` for `s3` + `minio`
  - `GCSProvider` for `gs`

Migrate both callers to shared open path:

- `backend/src/v2/objectstore/OpenBucket()` delegates opening.
- `backend/src/apiserver/client_manager/openBucketWithRetry()` delegates opening.

Keep existing transfer code and storage APIs unchanged.

## Design Details

### Core Provider Contract

```go
type Provider interface {
    Name() string
    Capabilities() CapabilityMetadata
    OpenBucket(ctx context.Context, request *OpenRequest) (*blob.Bucket, error)
}
```

### Shared Open Request/Config

```go
type OpenRequest struct {
    Namespace string
    K8sClient kubernetes.Interface
    Config    *Config
}

type Config struct {
    Scheme      string
    BucketName  string
    Prefix      string
    QueryString string
    SessionInfo *SessionInfo
}
```

`SessionInfo.Params` keeps existing key conventions:

- S3 family: `fromEnv`, `secretName`, `accessKeyKey`, `secretKeyKey`, `region`, `endpoint`, `disableSSL`,
  `forcePathStyle`, `maxRetries`.
- GCS: `fromEnv`, `secretName`, `tokenKey`.
- API server compatibility: inline `accessKey` / `secretAccessKey` mapping.

### Registry and Dispatch

```go
registry := NewDefaultRegistry()
registry.Register(&S3Provider{}, "s3", "minio")
registry.Register(&GCSProvider{}, "gs", "gcs")
```

Dispatch logic:

1. If `SessionInfo == nil`: fallback to URL open (`blob.OpenBucket`) with current query behavior.
2. Else: resolve provider by `SessionInfo.Provider`, call provider-specific opener.

### Phase 1 Provider Behavior

#### S3Provider

- Supports three auth modes:
  1. env/default credential chain;
  2. K8s secret-backed keys;
  3. inline static keys (API server compatibility path).
- Uses AWS SDK v2 + `s3blob.OpenBucketV2` in provider-session mode.
- Preserves path-style and endpoint behavior for S3-compatible backends.

#### GCSProvider

- Supports env/default creds or secret-backed token JSON.
- Uses `gcsblob.OpenBucket` and prefixed buckets.

### Caller Integration Snippets

#### Runtime integration

```go
commonConfig := &commonobjectstore.Config{
    Scheme:      config.Scheme,
    BucketName:  config.BucketName,
    Prefix:      config.Prefix,
    QueryString: config.QueryString,
}
openedBucket, err := commonobjectstore.OpenBucket(ctx, &commonobjectstore.OpenRequest{
    Namespace: namespace,
    K8sClient: k8sClient,
    Config:    commonConfig,
})
```

#### API server integration

```go
sharedConfig := toSharedObjectStoreConfig(config)
bucket, err = commonobjectstore.OpenBucket(ctx, &commonobjectstore.OpenRequest{
    Config: sharedConfig,
})
```

### Data Flow

```mermaid
flowchart LR
    subgraph callers [KFPCallers]
      apiServer[ApiServer]
      driver[Driver]
      launcher[Launcher]
    end

    subgraph sharedLayer [SharedObjectStoreLayer]
      openBucket[OpenBucket]
      registry[ProviderRegistry]
      s3Provider[S3Provider]
      gcsProvider[GCSProvider]
    end

    blobBucket[BlobBucket]

    apiServer --> openBucket
    driver --> openBucket
    launcher --> openBucket
    openBucket --> registry
    registry --> s3Provider
    registry --> gcsProvider
    s3Provider --> blobBucket
    gcsProvider --> blobBucket
```

## Phase-by-Phase Plan

## Phase 0: Baseline and Safety Lock

### Scope

- Validate current behavior before abstraction changes.

### Required Checks

- `go test ./backend/src/apiserver/client_manager`
- `go test ./backend/src/v2/objectstore`

### Exit Criteria

- Existing tests pass and become baseline regression guardrail.

## Phase 1: Shared Layer Introduction

### Scope

- Add shared package and providers (`s3/minio`, `gs`).
- Add registry and request/config model.

### Files

- `backend/src/common/objectstore/types.go`
- `backend/src/common/objectstore/registry.go`
- `backend/src/common/objectstore/open_bucket.go`
- `backend/src/common/objectstore/s3_provider.go`
- `backend/src/common/objectstore/gcs_provider.go`

### Constraints

- Keep behavior of existing session params.
- Keep URL fallback behavior for configs that rely on query params.

### Exit Criteria

- Shared package tests pass.

## Phase 2: Runtime Migration (Driver/Launcher Open Path)

### Scope

- Replace provider-switch opening in `backend/src/v2/objectstore/OpenBucket()` with shared dispatch.
- Leave upload/download implementations untouched.

### Files

- `backend/src/v2/objectstore/object_store.go`

### Constraints

- No changes to:
  - `UploadBlob`;
  - `DownloadBlob`;
  - path traversal protections;
  - `KeyFromURI` and prefix semantics.

### Exit Criteria

- `go test ./backend/src/v2/objectstore` passes unchanged behavior.

## Phase 3: API Server Migration

### Scope

- Route API server bucket open through shared layer.
- Preserve `OBJECTSTORECONFIG_*` compatibility via mapper.

### Files

- `backend/src/apiserver/client_manager/client_manager.go`
- `backend/src/apiserver/client_manager/client_manager_test.go`

### Constraints

- Keep:
  - `ensureBucketExists` best-effort behavior;
  - retry behavior around opening;
  - `BlobObjectStore` API and object key layout.

### Exit Criteria

- `go test ./backend/src/apiserver/client_manager` passes.

## Phase 4: Extensibility Proof (Design + Scaffold)

### Scope

- Add provider conformance test strategy.
- Document Azure and PVC/file enablement model and constraints.

### Constraints

- No production enablement of `azblob://` or `file://` in Phase 1.

### Exit Criteria

- Proposal and test plan fully describe extension contract.

## Impact Analysis

### Backend Impact

1. **API server**: opening logic is delegated, but startup and object operations remain unchanged.
2. **driver/launcher**: opening logic is delegated, artifact transfer behavior remains unchanged.
3. **shared common package**: new reusable abstraction and provider implementations.

### Operational Impact

1. No required config migration in Phase 1.
2. Existing providers and default deployment assumptions remain valid.
3. Future provider onboarding complexity moves from cross-cutting edits to provider-specific implementation.

### Performance Impact

- No expected material regression in steady-state artifact transfer; only bucket opening path is centralized.

## Test Plan

### Unit Tests (Required)

1. Shared layer:
   - registry alias resolution;
   - request validation;
   - unsupported provider behavior;
   - URL fallback behavior.
2. API server:
   - env vs inline credentials mapping into shared config.
3. Existing packages:
   - `backend/src/apiserver/client_manager`;
   - `backend/src/v2/objectstore`.

### Command Set

- `go test ./backend/src/common/objectstore`
- `go test ./backend/src/v2/objectstore`
- `go test ./backend/src/apiserver/client_manager`

### Future Conformance Tests

Add a provider contract suite for:

- open semantics for env-backed auth;
- secret-backed auth error behavior;
- prefix handling and unsupported-provider behavior.

## Migration Strategy

No user migration required in Phase 1:

- `s3://`, `minio://`, `gs://` remain supported;
- API server `OBJECTSTORECONFIG_*` remains supported;
- runtime provider/session config conventions remain supported.

Future `azblob://` and `file://` additions are additive.

## Frontend Considerations

No frontend contract changes in Phase 1. Artifact URL handling and API responses remain unchanged.

## KFP Local Considerations

No immediate local behavior changes. `file://` backend can be introduced later for local/PVC scenarios once mount and
capability constraints are explicitly modeled and tested.

## Risks and Mitigations

1. **Credential handling drift**
   - Mitigation: centralize provider auth logic and add mapping tests.
2. **Behavior drift in existing providers**
   - Mitigation: keep transfer code unchanged and run existing package tests.
3. **Over-generalizing PVC as object storage**
   - Mitigation: treat PVC as mount-aware `file://` backend with capability flags and deployment constraints.
4. **Endpoint/SSL/path-style compatibility regressions on S3-compatible stores**
   - Mitigation: preserve existing params (`endpoint`, `disableSSL`, `forcePathStyle`, `maxRetries`) and add focused tests.

## Open Issues and Follow-Ups

1. Define whether `azblob` auth in KFP should be env-first only or support secret-ref parity with current providers.
2. Define runtime policy for `file://` backend in multi-replica API server deployments.
3. Decide whether to add a provider capability gate at compile-time vs runtime config.

## Alternatives Considered

1. Keep per-service provider logic.
   - Rejected: duplicated behavior and long-term drift risk.
2. Full storage rewrite in a single phase.
   - Rejected: unnecessary risk vs no-regression requirement.
3. Migrate only one side (API server or runtime).
   - Rejected: does not achieve full-stack consistency.

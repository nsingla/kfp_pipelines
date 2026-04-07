# KEP: PVC/Filesystem Storage Backend (`file://`) for Kubeflow Pipelines

## Summary

This proposal describes adding a PVC/filesystem-backed storage backend (`file://`) to Kubeflow Pipelines, building on
the extensible provider model introduced in [KEP-13080](../13080-storage-backend-abstraction/README.md). Unlike object
storage backends (`s3://`, `gs://`, `azblob://`), PVC storage operates under fundamental Kubernetes constraints around
access modes, namespace isolation, and multi-node scheduling that make it unsuitable as a general-purpose drop-in
replacement for object storage.

This proposal documents those constraints explicitly, defines the deployment scenarios where `file://` is viable, and
specifies the requirements for a safe, limited-scope implementation.

**Primary use case**: air-gapped or on-premises environments where no object storage service is available and users
accept the trade-offs documented here.

## Motivation

Several KFP users have requested PVC-based artifact storage as an alternative to object storage:

- **Air-gapped environments**: clusters with no access to S3, GCS, MinIO, or Azure Blob
  ([kubeflow/pipelines#10510](https://github.com/kubeflow/pipelines/issues/10510)).
- **Cost-sensitive on-premises deployments**: users who already have shared filesystem infrastructure (NFS, CephFS) and
  do not want to operate an additional object storage service.
- **Local development**: single-node clusters (kind, minikube) where a hostPath or local PVC is simpler than running
  MinIO.

Go CDK supports filesystem buckets via `gocloud.dev/blob/fileblob` using the `file://` scheme. This fits PVC
conceptually, but the operational constraints are significant and must be modeled explicitly.

## Goals

1. Enable `file://` backend for artifact storage in single-user, single-namespace deployments.
2. Document all Kubernetes constraints that limit PVC viability as a storage backend.
3. Define capability metadata that callers can use to gate operations the backend does not support.
4. Provide a clear deployment viability matrix so operators can assess fit before enabling.
5. Ensure no regression for existing object storage backends.

## Non-Goals

1. Making `file://` a general-purpose replacement for object storage.
2. Solving cross-namespace PVC sharing at the Kubernetes level.
3. Enabling `file://` in multi-user (Profile-based) deployments without explicit NFS or RWX infrastructure.
4. Adding HTTP-based artifact serving (pre-signed URLs) for filesystem backends.
5. Modifying the KFP frontend to mount or read PVCs directly.

## Requirements and Constraints

### Kubernetes PVC Access Modes

Kubernetes defines four PVC access modes. Each has distinct implications for KFP:

| Access Mode | Abbrev. | Node Scope | Pod Scope | Read | Write | K8s Stability |
|---|---|---|---|---|---|---|
| ReadWriteOnce | RWO | Single node | Multiple pods (same node) | Yes | Yes | GA |
| ReadOnlyMany | ROX | Multiple nodes | Multiple pods | Yes | No | GA |
| ReadWriteMany | RWX | Multiple nodes | Multiple pods | Yes | Yes | GA |
| ReadWriteOncePod | RWOP | Single node | Single pod | Yes | Yes | GA (v1.29+) |

**Key nuances:**

- **RWO restricts by node, not by pod.** Multiple pods on the same node can all read and write the volume. This is a
  common misconception.
- **RWOP** (GA in Kubernetes v1.29+) is the only mode that restricts to exactly one pod cluster-wide. It requires CSI
  volumes.
- **Access modes are advisory, not enforced at the storage layer.** Kubernetes does not enforce write protection once
  storage is mounted. A volume created as ROX is not guaranteed to be read-only by the underlying storage.

### KFP Access Mode Requirements

KFP's architecture requires concurrent read/write access from multiple pods:

1. **API server** (potentially multi-replica) reads and writes artifacts.
2. **Pipeline steps** (driver/launcher pods in user namespaces) write artifacts during execution and read artifacts from
   upstream steps.
3. **UI server** reads artifacts for visualization.

This means:

- **RWO is only viable if all pods are scheduled on the same node** — not viable for production multi-node clusters.
- **ROX cannot be the primary store** — pipeline steps must write artifacts.
- **RWX is the minimum requirement** for any multi-node deployment.
- **RWOP is only viable for single-replica API server with sequential (non-parallel) pipeline steps** — not practical.

### RWX Storage Availability by Cloud Provider

**No cloud provider's block storage supports RWX.** Every cloud requires a separate, more expensive file-storage
service:

| Cloud Provider | Block Storage (RWO only) | File Storage (RWX capable) | CSI Driver |
|---|---|---|---|
| **AWS** | EBS (`ebs.csi.aws.com`) | EFS (`efs.csi.aws.com`) | EFS adds ms-level latency vs EBS; elastic capacity |
| **GCP** | Persistent Disk (`pd.csi.storage.gke.io`) | Filestore (`filestore.csi.storage.gke.io`) | Basic tier has **1 TB minimum** — significant cost floor |
| **Azure** | Azure Disk (`disk.csi.azure.com`) | Azure Files (`file.csi.azure.com`) | Simplest RWX option; SMB or NFS protocol |
| **Bare metal** | Rook-Ceph RBD, Longhorn | NFS, Rook-CephFS | Self-managed infrastructure required |

**Performance characteristics of RWX vs object storage:**

- Network filesystems (NFS, CephFS, EFS) add per-operation latency compared to direct block storage.
- ML pipeline artifacts are often many small files. Network filesystems have higher per-operation overhead for this
  pattern than object storage.
- AWS EFS: several milliseconds of latency per operation vs sub-millisecond for EBS. Throughput scales well (up to
  1,500 MiBps) but per-operation latency is noticeably higher.
- GCP Filestore basic HDD tier has lower IOPS than SSD; enterprise tier supports multi-share but at higher cost.

### Cross-Namespace PVC Isolation (Multi-User Blocker)

**PVCs are fundamentally namespace-scoped in Kubernetes.** This is the single largest constraint for KFP's multi-user
deployment model.

In KFP multi-user mode:

- Each user gets a **Profile**, which maps 1:1 to a Kubernetes **namespace**.
- Pipeline steps run in the user's namespace.
- The API server runs in the `kubeflow` namespace.
- The profile controller creates per-namespace artifact path prefixes and per-namespace credentials.

**PVC implications:**

- A PVC in namespace `profile-alice` **cannot** be mounted by a pod in namespace `profile-bob`. This is by Kubernetes
  design for security isolation.
- The API server in `kubeflow` namespace **cannot** mount PVCs from user namespaces.
- Pipeline steps in user namespaces **cannot** mount a central PVC from the `kubeflow` namespace.
- The current isolation model (per-namespace S3 path prefixes + per-namespace IAM credentials) has **no PVC
  equivalent**.

**Workarounds exist but all add significant complexity:**

1. **NFS server with cross-namespace service exposure**: Deploy an NFS server and create separate PVCs in each namespace
   pointing to the same NFS export with different subdirectories. Requires NFS infrastructure and per-namespace PVC
   provisioning.
2. **NetApp Astra Trident subordinate volumes**: Supports cross-namespace volume sharing as a first-class feature, but
   snapshots, clones, and mirroring are not possible on subordinate volumes. Vendor-specific.
3. **`CrossNamespaceVolumeDataSource` feature gate** (alpha since v1.26): Allows a PVC to reference a data source in
   another namespace via `ReferenceGrant` CRDs. However, this creates **clones/snapshots**, not live shared access.
   Not suitable for artifact storage.
4. **Multiple PVCs backed by the same storage**: Driver-specific (e.g. NFS CSI), not a portable Kubernetes-native
   pattern.

**Recommendation**: `file://` backend in multi-user mode should be documented as requiring NFS or equivalent
cross-namespace infrastructure. Without it, `file://` is limited to single-user or single-namespace deployments.

### UI Artifact Serving Limitation

Object storage provides HTTP-based access (pre-signed URLs, REST APIs). The KFP frontend uses these to serve artifact
visualizations. PVC filesystems have no equivalent.

For `file://` backends, the UI cannot serve artifacts unless:

- The UI pod mounts the same PVC (requires co-location or RWX).
- A sidecar/proxy is deployed to serve filesystem contents over HTTP.
- An artifact-serving component is added that mounts the PVC and exposes an HTTP API.

This is a known gap tracked in
[kubeflow/pipelines#1497](https://github.com/kubeflow/pipelines/issues/1497).

### Operational Gaps vs Object Storage

| Capability | Object Storage (S3/GCS/Azure) | PVC/Filesystem |
|---|---|---|
| HTTP API / pre-signed URLs | Yes | No |
| Atomic PUT/GET | Yes | No (race conditions possible on RWX) |
| Elastic capacity | Yes | No (fixed size, manual resize required*) |
| Built-in versioning | Yes | No |
| Cross-region replication | Yes | No (requires Velero or similar) |
| Event notifications | Yes | No |
| Metadata / tagging | Yes | No |
| Namespace-agnostic access | Yes (credential-based) | No (namespace-scoped) |
| Multi-replica API server | Yes | Only with RWX |
| Concurrent write safety | Yes (per-object atomicity) | Application must handle locking |

\* Exception: AWS EFS provides elastic capacity.

## Deployment Viability Matrix

| Deployment Scenario | `file://` Viable? | Access Mode Required | Notes |
|---|---|---|---|
| Single-user, single-node (dev) | **Yes** | RWO sufficient | Simplest case; hostPath or local PV works |
| Single-user, multi-node | **Conditional** | RWX required | Requires RWX-capable StorageClass |
| Multi-user (Profiles) | **Very limited** | RWX + NFS/cross-NS | Requires NFS or equivalent; no native PVC cross-namespace sharing |
| Multi-replica API server | **Conditional** | RWX required | Must accept consistency trade-offs |
| Air-gapped, no object store | **Best-effort** | Depends on topology | Must accept UI artifact and performance limitations |

## Design Details

### Provider Implementation

The `file://` provider plugs into the registry model from KEP-13080:

```go
registry.Register(&FileProvider{}, "file")
```

### FileProvider

```go
type FileProvider struct{}

func (p *FileProvider) Name() string { return "file" }

func (p *FileProvider) Capabilities() CapabilityMetadata {
    return CapabilityMetadata{
        SupportsHTTPAccess:     false,
        SupportsPreSignedURLs:  false,
        SupportsCrossNamespace: false,
        SupportsMultiReplica:   false, // conservative default; true only if RWX confirmed
        SupportsElasticCapacity: false,
        SupportsVersioning:     false,
        RequiresVolumeMount:    true,
    }
}

func (p *FileProvider) OpenBucket(ctx context.Context, req *OpenRequest) (*blob.Bucket, error) {
    // Uses gocloud.dev/blob/fileblob
    // req.Config.BucketName maps to the filesystem root path (mount point)
    // req.Config.Prefix maps to subdirectory within the mount
}
```

### Capability Metadata Extension

KEP-13080's `CapabilityMetadata` should be extended for `file://`:

```go
type CapabilityMetadata struct {
    SupportsHTTPAccess      bool   // false for file://
    SupportsPreSignedURLs   bool   // false for file://
    SupportsCrossNamespace  bool   // false for file:// without NFS
    SupportsMultiReplica    bool   // only if RWX available
    SupportsElasticCapacity bool   // false for most PVCs (true for EFS)
    SupportsVersioning      bool   // false for file://
    RequiresVolumeMount     bool   // true for file://
}
```

Callers (API server, driver, launcher, UI) must check capabilities before attempting operations that the backend does
not support. For example, the UI should gracefully degrade artifact visualization when `SupportsHTTPAccess` is false.

### Volume Mount Configuration

The `file://` backend requires that the PVC is mounted into the pods that need access. This is a deployment-time
concern, not a runtime concern:

1. **API server Deployment**: must include a `volumeMount` for the artifact PVC.
2. **Pipeline step pods**: the driver/launcher must add `volumeMount` specs to pipeline step pod templates.
3. **UI server Deployment**: must include a `volumeMount` if artifact visualization is desired.

Configuration via environment variables (consistent with existing `OBJECTSTORECONFIG_*` pattern):

```
OBJECTSTORECONFIG_SCHEME=file
OBJECTSTORECONFIG_BUCKETNAME=/mnt/artifacts    # mount path
OBJECTSTORECONFIG_PREFIX=pipeline-artifacts    # subdirectory
```

### Namespace Isolation Strategy (Multi-User)

For multi-user deployments that choose to use `file://` with NFS:

1. A single NFS export is mounted in all relevant namespaces.
2. Per-namespace subdirectories provide logical isolation (e.g., `/mnt/artifacts/private-artifacts/{namespace}/`).
3. Isolation is **path-based, not credential-based** — the filesystem does not enforce access boundaries between
   namespaces. This is a weaker isolation guarantee than object storage with per-namespace IAM credentials.
4. The profile controller would need to be extended to create per-namespace subdirectories and potentially set
   filesystem permissions (UID/GID) per namespace.

**This is a fundamentally weaker security model than object storage.** Operators must understand and accept this
trade-off.

## Phase Plan

### Phase A: Single-User FileProvider (Minimum Viable)

**Scope:**
- Implement `FileProvider` with `gocloud.dev/blob/fileblob`.
- Register `file` scheme in the provider registry.
- Add volume mount configuration to API server deployment.
- Add volume mount injection for pipeline step pods.

**Constraints:**
- Single-user mode only.
- UI artifact visualization not supported (graceful degradation).
- RWX StorageClass required for multi-node clusters.

**Exit Criteria:**
- `go test ./backend/src/common/objectstore` passes with `file` provider.
- End-to-end pipeline execution with artifact upload/download works on a single-node cluster.

### Phase B: Multi-Node Support

**Scope:**
- Validate and document RWX StorageClass requirements.
- Add startup validation that checks PVC access mode and warns if RWO on multi-node.
- Add volume mount injection for driver/launcher pods.

**Exit Criteria:**
- Pipeline execution works on multi-node cluster with RWX PVC.
- Clear error/warning when RWX is not available.

### Phase C: Multi-User Support (Requires NFS)

**Scope:**
- Extend profile controller to create per-namespace subdirectories.
- Add path-based namespace isolation to `FileProvider`.
- Document NFS infrastructure requirements.
- Add capability checks to UI for graceful degradation.

**Constraints:**
- Requires NFS or equivalent cross-namespace storage.
- Security isolation is path-based only.

**Exit Criteria:**
- Multi-user pipeline execution with namespace-isolated artifacts.
- Documentation covers NFS setup and security trade-offs.

## Test Plan

### Unit Tests

1. `FileProvider` open/close semantics.
2. Capability metadata correctness.
3. Path construction and prefix handling.
4. Namespace subdirectory isolation logic.
5. Graceful behavior when volume is not mounted.

### Integration Tests

1. Single-node artifact upload/download via `file://`.
2. Multi-pod concurrent read/write on RWX volume.
3. API server + pipeline step artifact handoff.
4. Graceful degradation when capabilities are not met.

### Manual Validation Matrix

| Scenario | Storage | Expected Result |
|---|---|---|
| Single-node, RWO PVC | hostPath / local | Pass |
| Multi-node, RWO PVC | EBS / PD | Fail with clear error |
| Multi-node, RWX PVC | EFS / Filestore / NFS | Pass |
| Multi-user, no NFS | Any | Fail with clear error |
| Multi-user, NFS | NFS export | Pass with isolation caveats |

## Risks and Mitigations

1. **Operators enable `file://` without RWX, causing silent data loss.**
   - Mitigation: startup validation checks PVC access mode; log warning or block startup if RWO on multi-node cluster.

2. **Path-based isolation is weaker than credential-based isolation.**
   - Mitigation: document clearly; require explicit opt-in for multi-user; consider filesystem UID/GID enforcement.

3. **Race conditions on concurrent writes to shared filesystem.**
   - Mitigation: use atomic write patterns (write-to-temp + rename) where possible; document limitation.

4. **PVC capacity exhaustion with no elastic scaling.**
   - Mitigation: add monitoring/alerting guidance; document capacity planning.

5. **UI cannot serve artifacts.**
   - Mitigation: graceful degradation; document limitation; future work for artifact proxy sidecar.

6. **Performance regression for small-file-heavy workloads.**
   - Mitigation: document performance characteristics; recommend object storage for production workloads.

## Alternatives Considered

1. **S3-FUSE (Mountpoint for Amazon S3, GCS FUSE).**
   - Mounts object storage as a filesystem. Avoids PVC constraints entirely. However, S3-FUSE has known limitations:
     no random writes, no hard links, no file locking, eventual consistency for metadata. Not suitable as a general
     solution.

2. **Embedded MinIO per namespace.**
   - Deploy a MinIO instance per user namespace. Provides full object storage semantics but adds operational overhead
     (one StatefulSet per namespace, more resources, more maintenance).

3. **Do nothing — require object storage.**
   - The simplest option. However, this blocks adoption in air-gapped and cost-constrained environments.

## Open Issues

1. Define the artifact proxy/sidecar design for UI artifact serving with `file://` backend.
2. Determine whether filesystem UID/GID enforcement per namespace is practical for multi-user isolation.
3. Evaluate S3-FUSE as a potential alternative that avoids PVC constraints while keeping filesystem semantics.
4. Define capacity monitoring and alerting guidance for PVC-based artifact storage.
5. Determine minimum Kubernetes version requirement (RWOP requires v1.29+, CrossNamespaceVolumeDataSource requires
   v1.26+ with alpha gate).

## References

- [KEP-13080: Extensible Multi-Backend Object Storage](../13080-storage-backend-abstraction/README.md) — prerequisite
  shared provider abstraction.
- [kubeflow/pipelines#10510](https://github.com/kubeflow/pipelines/issues/10510) — Support PV/PVCs as alternative to
  Object Storage.
- [kubeflow/pipelines#1497](https://github.com/kubeflow/pipelines/issues/1497) — Web UI should support get artifact
  from local path.
- [Kubernetes PVC Access Modes](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#access-modes)
- [Go CDK fileblob](https://pkg.go.dev/gocloud.dev/blob/fileblob) — filesystem blob implementation.

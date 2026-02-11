# Talos Operator - Module by Module Guide

This document provides a comprehensive explanation of the talos-operator project, breaking down each module and its responsibilities.

## Overview

The talos-operator is a Kubernetes operator that enables declarative management of Talos Linux-based Kubernetes clusters. It uses the operator pattern to automate the creation, configuration, and lifecycle management of Talos clusters in different environments (bare metal, containers, and eventually cloud).

## Architecture Overview

```
talos-operator/
├── api/              # Custom Resource Definitions (CRDs)
├── cmd/              # Main application entry points
├── internal/         # Internal controllers and business logic
├── pkg/              # Reusable packages and utilities
├── config/           # Kubernetes manifests and configuration
├── deploy/           # Helm charts for deployment
├── examples/         # Example CRD manifests
├── docs/             # Documentation
├── hack/             # Development and build scripts
├── test/             # Test suites
└── ui/               # Web UI for the operator
```

---

## 1. API Module (`api/v1alpha1/`)

**Purpose**: Defines the custom Kubernetes resources (CRDs) that users interact with to declare their desired Talos cluster state.

### Core Resource Types

#### 1.1 TalosCluster (`taloscluster_types.go`)
- **What it is**: Top-level resource representing a complete Talos Kubernetes cluster
- **Key fields**:
  - `ControlPlane` or `ControlPlaneRef`: Defines or references the control plane configuration
  - `Worker` or `WorkerRef`: Defines or references the worker node configuration
- **Design philosophy**: Supports both inline configuration and references to separate resources for flexibility
- **Use case**: Single manifest to define an entire cluster, or use references for more modular design

#### 1.2 TalosControlPlane (`taloscontrolplane_types.go`)
- **What it is**: Represents the Kubernetes control plane (API server, etcd, scheduler, controller-manager)
- **Key fields**:
  - `Version`: Talos version (e.g., "v1.10.3")
  - `KubeVersion`: Kubernetes version (e.g., "v1.33.1")
  - `Mode`: Installation mode ("container" or "metal")
  - `Replicas`: Number of control plane nodes
  - `MetalSpec`: Machine configurations for bare metal deployments
  - `Endpoint`: Kubernetes API server endpoint
  - `ClusterDomain`: DNS domain for the cluster
  - `PodCIDRs`, `ServiceCIDRs`: Network configuration
  - `CNI`: Container Network Interface configuration
- **Immutable fields**: ClusterDomain, Mode (cannot be changed after creation)
- **Validation**: Ensures version upgrades only, required fields based on mode

#### 1.3 TalosWorker (`talosworker_types.go`)
- **What it is**: Represents worker nodes in the Talos cluster
- **Key fields**:
  - `Version`: Talos version
  - `KubeVersion`: Kubernetes version (derived from control plane)
  - `Mode`: Installation mode ("container" or "metal")
  - `Replicas`: Number of worker nodes
  - `MetalSpec`: Machine configurations for bare metal
  - `ControlPlaneRef`: Reference to the associated control plane
  - `ConfigRef`: Optional reference to custom configuration
- **Relationship**: Always associated with a TalosControlPlane
- **Use case**: Scale worker nodes independently from control plane

#### 1.4 TalosMachine (`talosmachine_types.go`)
- **What it is**: Represents individual machines (physical or virtual) in the cluster
- **Key fields**:
  - `Type`: Machine type ("controlplane" or "worker")
  - `Hostname`: Machine hostname
  - `IPs`: Network addresses
  - `InstallDisk`: Disk for Talos installation
- **Use case**: Low-level machine management, usually created by controllers

#### 1.5 TalosClusterAddon (`talosclusteraddon_types.go`)
- **What it is**: Represents add-ons to be installed in the cluster (e.g., CNI, CSI, monitoring)
- **Key fields**:
  - `HelmChartSpec`: Helm chart details for the addon
  - `TargetNamespace`: Where to install the addon
  - `Values`: Helm values for customization
- **Use case**: Declaratively manage cluster add-ons

#### 1.6 TalosClusterAddonRelease (`talosclusteraddonrelease_types.go`)
- **What it is**: Represents a specific release/version of a cluster addon
- **Purpose**: Track addon installations and upgrades
- **Relationship**: Created by TalosClusterAddon controller

#### 1.7 TalosEtcdBackup (`talosetcdbackup_types.go`)
- **What it is**: Represents a backup of the etcd database
- **Key fields**:
  - `ClusterRef`: Reference to the cluster to backup
  - `StorageType`: Where to store backup (e.g., S3)
  - `Retention`: Backup retention policy
- **Use case**: Create on-demand etcd backups for disaster recovery

#### 1.8 TalosEtcdBackupSchedule (`talosetcdbackupschedule_types.go`)
- **What it is**: Defines a schedule for automatic etcd backups
- **Key fields**:
  - `Schedule`: Cron expression for backup timing
  - `BackupTemplate`: Template for backup configuration
- **Use case**: Automated, scheduled etcd backups

#### 1.9 Common Types (`common.go`)
- **What it is**: Shared types and structures used across multiple CRDs
- **Examples**:
  - `MetalSpec`: Configuration for bare metal machines
  - `CNISpec`: Container Network Interface configuration
  - `ResourceRequirements`: CPU/Memory specifications
  - `TalosConfigPatch`: Custom Talos configuration patches

---

## 2. Internal Module (`internal/controller/`)

**Purpose**: Contains the reconciliation logic (controllers) that watches CRDs and makes the actual state match the desired state.

### Controller Components

#### 2.1 TalosCluster Controller (`taloscluster_controller.go`)
- **Responsibility**: Orchestrates the creation and management of complete Talos clusters
- **Key operations**:
  - Creates/updates TalosControlPlane and TalosWorker resources (if inline)
  - Manages cluster-level configuration (secrets, configmaps)
  - Coordinates control plane and worker reconciliation
  - Updates cluster status and conditions
- **Reconciliation flow**:
  1. Validate cluster specification
  2. Generate or retrieve Talos secrets bundle
  3. Create/update control plane resource
  4. Create/update worker resources
  5. Wait for cluster components to be ready
  6. Update cluster status

#### 2.2 TalosControlPlane Controller (`taloscontrolplane_controller.go`)
- **Responsibility**: Manages the Kubernetes control plane components
- **Key operations**:
  - Generates Talos control plane configuration
  - Creates/manages control plane machines
  - Bootstraps etcd cluster
  - Configures Kubernetes API server, controller-manager, scheduler
  - Handles Kubernetes version upgrades
  - Generates and stores kubeconfig
- **Mode-specific logic**:
  - **Container mode**: Creates Kubernetes Pods for control plane nodes
  - **Metal mode**: Applies configuration to physical/virtual machines
- **Complex scenarios**:
  - Initial cluster bootstrap
  - Node replacement
  - Version upgrades (coordinated rolling updates)
  - etcd membership management

#### 2.3 TalosWorker Controller (`talosworker_controller.go`)
- **Responsibility**: Manages worker nodes in the cluster
- **Key operations**:
  - Generates Talos worker configuration
  - Creates/manages worker machines
  - Joins workers to the cluster
  - Handles worker scaling (up/down)
  - Manages worker upgrades
- **Mode-specific logic**:
  - **Container mode**: Creates Kubernetes Pods for worker nodes
  - **Metal mode**: Applies configuration to physical/virtual machines
- **Integration**: Retrieves control plane information to join workers correctly

#### 2.4 TalosMachine Controller (`talosmachine_controller.go`)
- **Responsibility**: Manages individual machine lifecycle
- **Key operations**:
  - Applies Talos configuration to machines
  - Monitors machine health and status
  - Handles machine updates and reboots
  - Manages machine deletion and cleanup
- **Direct interaction**: Uses Talos API to communicate with machines
- **State management**: Tracks machine readiness, version, and conditions

#### 2.5 TalosClusterAddon Controller (`talosclusteraddon_controller.go`)
- **Responsibility**: Manages cluster add-ons using Helm
- **Key operations**:
  - Installs Helm charts into target cluster
  - Manages addon upgrades and configuration changes
  - Creates TalosClusterAddonRelease resources
  - Monitors addon health
- **Integration**: Uses Helm client to interact with target clusters
- **Use cases**: Installing CNI plugins, CSI drivers, monitoring stacks

#### 2.6 TalosClusterAddonRelease Controller (`talosclusteraddonrelease_controller.go`)
- **Responsibility**: Tracks and manages specific addon release instances
- **Key operations**:
  - Records addon installation status
  - Manages release lifecycle
  - Handles rollbacks if needed
- **Relationship**: Child resource of TalosClusterAddon

#### 2.7 TalosEtcdBackup Controller (`talosetcdbackup_controller.go`)
- **Responsibility**: Performs etcd database backups
- **Key operations**:
  - Connects to etcd via Talos API
  - Creates etcd snapshots
  - Uploads backups to configured storage (S3, etc.)
  - Updates backup status and metadata
- **Error handling**: Retries on failure, reports status

#### 2.8 TalosEtcdBackupSchedule Controller (`talosetcdbackupschedule_controller.go`)
- **Responsibility**: Schedules automatic etcd backups
- **Key operations**:
  - Parses cron schedule
  - Creates TalosEtcdBackup resources on schedule
  - Manages backup retention and cleanup
  - Handles schedule updates
- **Implementation**: Uses controller-runtime's scheduled reconciliation

### Helper Components

#### 2.9 Container Mode Common (`container_mode_common.go`)
- **Purpose**: Shared logic for managing Talos nodes in container mode
- **Functions**:
  - Creating Kubernetes Pods that run Talos
  - Configuring networking for container nodes
  - Managing storage for container nodes
  - Handling pod lifecycle

#### 2.10 Machine Common (`machine_common.go`)
- **Purpose**: Shared utilities for machine management
- **Functions**:
  - Common machine operations
  - Machine state validation
  - Helper functions for machine controllers

#### 2.11 Constants (`constants.go`)
- **Purpose**: Defines constants used across controllers
- **Examples**: Label keys, annotation keys, finalizers, default values

#### 2.12 Predicates (`predicate.go`)
- **Purpose**: Custom event filters for controllers
- **Usage**: Determines which events trigger reconciliation

---

## 3. Package Module (`pkg/`)

**Purpose**: Reusable libraries and utilities that provide core functionality to controllers.

### 3.1 Talos Package (`pkg/talos/`)
- **Purpose**: Abstracts Talos API interactions and configuration generation

#### Key Components:
- **`client.go`**: Talos API client wrapper
  - Creates authenticated connections to Talos nodes
  - Provides methods for common Talos operations
  - Handles connection pooling and retries

- **`bundle.go`**: Secret bundle management
  - Generates Talos secrets (CA, tokens, keys)
  - Marshals/unmarshals secret bundles
  - Stores secrets in Kubernetes Secrets

- **`control_plane.go`**: Control plane configuration generation
  - Generates Talos config for control plane nodes
  - Configures etcd, kube-apiserver, controller-manager, scheduler
  - Handles control plane-specific patches

- **`worker.go`**: Worker configuration generation
  - Generates Talos config for worker nodes
  - Configures kubelet and runtime
  - Handles worker-specific patches

- **`kubernetes.go`**: Kubernetes-specific operations
  - Cluster bootstrap
  - Version upgrades
  - Node operations

- **`etcd.go`**: etcd management
  - etcd cluster operations
  - Backup and restore
  - Member management

- **`write.go`**: Configuration application
  - Writes config to Talos nodes
  - Handles machine configuration updates
  - Manages reboots and drains

- **`metakey_tpl.go`**: Meta key templating
  - Templates for machine identification
  - Used in bare metal deployments

### 3.2 Storage Package (`pkg/storage/`)
- **Purpose**: Manages backup storage backends

#### Key Components:
- **`s3.go`**: S3-compatible storage implementation
  - Upload/download backups to S3
  - Supports various S3-compatible providers
  - Handles authentication and encryption

### 3.3 Helm Package (`pkg/helm/`)
- **Purpose**: Helm chart operations for addon management

#### Key Components:
- **`client.go`**: Helm client wrapper
  - Install/upgrade Helm releases
  - Manage Helm repositories
  - Handle release lifecycle

### 3.4 Utils Package (`pkg/utils/`)
- **Purpose**: General utility functions

#### Key Components:
- **`utils.go`**: Common utilities
  - String manipulation
  - Validation helpers
  - Conversion functions

---

## 4. Command Module (`cmd/`)

**Purpose**: Application entry points and CLI commands.

### 4.1 Main (`cmd/main.go`)
- **Primary function**: Starts the operator manager
- **Responsibilities**:
  - Initializes Kubernetes client
  - Registers all CRDs and controllers
  - Sets up metrics and health endpoints
  - Configures leader election
  - Handles graceful shutdown
  
- **Special command**: `upgrade-k8s`
  - Standalone command for upgrading Kubernetes versions
  - Can be run as a Job in the cluster
  - Handles coordinated control plane and worker upgrades

### Key initialization steps:
1. Parse flags and environment variables
2. Set up logging
3. Create controller manager
4. Register schemes and controllers
5. Start health and metrics servers
6. Run manager (blocks until shutdown)

---

## 5. Configuration Module (`config/`)

**Purpose**: Kubernetes manifests and kustomize configurations for deploying the operator.

### 5.1 CRD (`config/crd/`)
- Contains generated CustomResourceDefinition manifests
- Defines the API schema for all custom resources
- Includes validation rules and OpenAPI schemas

### 5.2 Manager (`config/manager/`)
- Deployment manifest for the operator manager
- Defines resource limits, security context
- Configures service account and RBAC

### 5.3 RBAC (`config/rbac/`)
- Role and RoleBinding definitions
- Grants necessary permissions to the operator
- Includes service account creation

### 5.4 Default (`config/default/`)
- Kustomization base combining all components
- Default configuration for deployment

### 5.5 Prometheus (`config/prometheus/`)
- ServiceMonitor for Prometheus integration
- Exposes operator metrics

### 5.6 Network Policy (`config/network-policy/`)
- NetworkPolicy definitions for security

### 5.7 Samples (`config/samples/`)
- Example CR manifests for testing

---

## 6. Deployment Module (`deploy/talos-operator/`)

**Purpose**: Helm chart for production deployment of the operator.

### Components:
- **Chart.yaml**: Helm chart metadata
- **values.yaml**: Configurable parameters
- **templates/**: Kubernetes manifest templates
  - Deployment, Service, RBAC resources
  - ConfigMaps for operator configuration

### Key configurations:
- Image repository and tag
- Resource requests and limits
- Replica count and scaling
- Environment variables
- Security settings

---

## 7. Examples Module (`examples/`)

**Purpose**: Provides ready-to-use example manifests demonstrating various deployment scenarios.

### Key examples:
- **`talos-cluster-container.yaml`**: Complete cluster in container mode
- **`talos-cluster-metal.yaml`**: Complete cluster on bare metal
- **`talos-controlplane-container.yaml`**: Control plane only (container)
- **`talos-controlplane-metal.yaml`**: Control plane only (bare metal)
- **`talos-worker-container.yaml`**: Workers only (container)
- **`talos-worker-metal.yaml`**: Workers only (bare metal)
- **`talos-controlplane-with-cni.yaml`**: Control plane with CNI configuration
- **`talos-cluster-addon.yaml`**: Addon installation example
- **`talos-etcd-backup.yaml`**: etcd backup example
- **`talos-etcd-backup-schedule.yaml`**: Scheduled backup example

---

## 8. Documentation Module (`docs/`)

**Purpose**: Comprehensive user and developer documentation.

### Structure:
- **`index.md`**: Main documentation landing page
- **`getting_started.md`**: Quick start guide
- **`crds/`**: Detailed CRD documentation
- **`operator_manual/`**: Operator operation guide
- **`diagram.md`**: Architecture diagrams
- **`kubernetes_upgrade_flow_diagram.md`**: Upgrade process flow
- **`metrics.md`**: Metrics and observability
- **`upgrading/`**: Version upgrade guides
- **`contributing.md`**: Contribution guidelines
- **`roadmap.md`**: Future plans

---

## 9. Hack Module (`hack/`)

**Purpose**: Development tools, scripts, and examples.

### 9.1 Talos Imager (`hack/talos-imager/`)
- Tool for creating custom Talos images
- Helps in bare metal deployments

### 9.2 Talos Client Examples (`hack/talos-client-examples/`)
- Example code for using Talos API directly
- Learning resources for Talos operations

### 9.3 Boilerplate (`hack/boilerplate.go.txt`)
- License header template for source files

---

## 10. Test Module (`test/`)

**Purpose**: Test suites for the operator.

### 10.1 E2E Tests (`test/e2e/`)
- End-to-end integration tests
- Tests complete cluster lifecycle
- Validates operator behavior in realistic scenarios

### 10.2 Test Utils (`test/utils/`)
- Helper functions for testing
- Test fixtures and mocks
- Shared test utilities

---

## 11. UI Module (`ui/`)

**Purpose**: Web-based user interface for the operator (optional component).

### Components:
- **Backend**: API server for UI data
- **Frontend**: Web interface for managing clusters
- **Features**:
  - Cluster visualization
  - Management operations
  - Status monitoring

---

## 12. Build and Development (`Makefile`, `magefile.go`)

**Purpose**: Build automation and development workflows.

### Key targets:
- **`make install`**: Install CRDs to cluster
- **`make deploy`**: Deploy operator to cluster
- **`make run`**: Run operator locally
- **`make test`**: Run unit tests
- **`make docker-build`**: Build container image
- **`make manifests`**: Generate CRD manifests
- **`make generate`**: Generate code (deepcopy, clients)

### Mage (magefile.go):
- Alternative build system using Go
- Provides programmatic build tasks
- Cross-platform build support

---

## Data Flow Overview

### Cluster Creation Flow:
1. User applies TalosCluster CR
2. TalosCluster controller:
   - Generates secret bundle
   - Creates TalosControlPlane CR
   - Creates TalosWorker CR (if defined)
3. TalosControlPlane controller:
   - Generates control plane config using `pkg/talos`
   - Creates TalosMachine CRs for each control plane node
   - Bootstraps first node
4. TalosMachine controller:
   - Applies config to machines using Talos API
   - Monitors machine status
5. TalosControlPlane controller:
   - Waits for etcd quorum
   - Generates kubeconfig
   - Stores in Secret
6. TalosWorker controller (if present):
   - Generates worker config
   - Creates TalosMachine CRs for workers
   - Joins workers to cluster
7. TalosCluster controller:
   - Updates status to Ready

### Backup Flow:
1. User creates TalosEtcdBackup or TalosEtcdBackupSchedule
2. Backup controller:
   - Connects to etcd via Talos API
   - Creates snapshot
   - Uses `pkg/storage` to upload to S3
   - Updates backup status

### Addon Installation Flow:
1. User creates TalosClusterAddon CR
2. Addon controller:
   - Retrieves kubeconfig from cluster
   - Uses `pkg/helm` to install chart
   - Creates TalosClusterAddonRelease CR
   - Monitors release status

---

## Key Design Patterns

### 1. Decoupled Resources
- Control plane and workers can be managed independently
- Enables flexible cluster topologies
- Supports references between resources

### 2. Mode-based Deployment
- **Container mode**: Kubernetes-in-Kubernetes (dev/testing)
- **Metal mode**: Physical/virtual machines (production)
- Same CRDs, different implementations

### 3. Configuration Generation
- Operator generates all Talos configs
- Users can provide patches for customization
- No manual Talos CLI commands needed

### 4. Kubernetes-native
- Everything stored in Kubernetes resources
- Uses standard controller patterns
- Integrates with existing K8s tooling

### 5. Gradual Upgrades
- Coordinated control plane upgrades
- Rolling worker upgrades
- Minimizes downtime

---

## Technology Stack

- **Language**: Go
- **Framework**: Kubebuilder / controller-runtime
- **APIs**: Talos Machine API, Kubernetes API
- **Storage**: S3-compatible backends
- **Package Management**: Helm 3
- **Build**: Make, Mage, Docker/Ko
- **Testing**: Ginkgo, controller-runtime test framework

---

## Summary

The talos-operator is a sophisticated Kubernetes operator that provides declarative management of Talos Linux clusters. Its modular architecture separates concerns clearly:

- **API module** defines what users want
- **Controller module** makes it happen
- **Package module** provides the tools
- **Command module** bootstraps everything
- **Config/Deploy modules** package it for users

This separation enables maintainability, testability, and extensibility while providing a powerful yet simple interface for managing Talos clusters at any scale.

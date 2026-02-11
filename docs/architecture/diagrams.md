# Architecture Diagrams

This page contains visual diagrams to help understand the talos-operator architecture.

## High-Level Architecture

```mermaid
graph TB
    subgraph "User Layer"
        User[Operator User]
        YAML[YAML Manifests]
    end
    
    subgraph "Kubernetes Control Plane"
        API[Kubernetes API Server]
        ETCD[(etcd)]
    end
    
    subgraph "Talos Operator"
        Manager[Operator Manager<br/>cmd/main.go]
        
        subgraph "Controllers"
            ClusterCtrl[TalosCluster<br/>Controller]
            CPCtrl[TalosControlPlane<br/>Controller]
            WorkerCtrl[TalosWorker<br/>Controller]
            MachineCtrl[TalosMachine<br/>Controller]
            BackupCtrl[TalosEtcdBackup<br/>Controller]
            AddonCtrl[TalosClusterAddon<br/>Controller]
        end
        
        subgraph "Packages"
            TalosPkg[pkg/talos<br/>Talos API Client]
            HelmPkg[pkg/helm<br/>Helm Client]
            StoragePkg[pkg/storage<br/>S3 Storage]
        end
    end
    
    subgraph "Managed Resources"
        TalosNodes[Talos Machines<br/>Physical/Virtual/Container]
        K8sCluster[Managed Kubernetes Clusters]
        S3[S3 Backup Storage]
    end
    
    User -->|kubectl apply| YAML
    YAML -->|Creates CRDs| API
    API <-->|Watches/Updates| Manager
    Manager --> ClusterCtrl
    ClusterCtrl --> CPCtrl
    ClusterCtrl --> WorkerCtrl
    CPCtrl --> MachineCtrl
    WorkerCtrl --> MachineCtrl
    Manager --> BackupCtrl
    Manager --> AddonCtrl
    
    MachineCtrl -->|Uses| TalosPkg
    BackupCtrl -->|Uses| TalosPkg
    BackupCtrl -->|Uses| StoragePkg
    AddonCtrl -->|Uses| HelmPkg
    
    TalosPkg -->|Talos API| TalosNodes
    TalosNodes -->|Runs| K8sCluster
    StoragePkg -->|Uploads| S3
    HelmPkg -->|Installs to| K8sCluster
```

## Resource Lifecycle Flow

```mermaid
sequenceDiagram
    participant User
    participant K8sAPI as Kubernetes API
    participant ClusterCtrl as TalosCluster Controller
    participant CPCtrl as ControlPlane Controller
    participant WorkerCtrl as Worker Controller
    participant MachineCtrl as Machine Controller
    participant TalosAPI as Talos API
    participant Machines as Talos Machines
    
    User->>K8sAPI: kubectl apply taloscluster.yaml
    K8sAPI->>ClusterCtrl: Watch event
    ClusterCtrl->>ClusterCtrl: Generate secrets bundle
    ClusterCtrl->>K8sAPI: Create TalosControlPlane CR
    ClusterCtrl->>K8sAPI: Create TalosWorker CR
    
    K8sAPI->>CPCtrl: Watch event
    CPCtrl->>CPCtrl: Generate control plane config
    CPCtrl->>K8sAPI: Create TalosMachine CRs (CP nodes)
    
    K8sAPI->>WorkerCtrl: Watch event
    WorkerCtrl->>WorkerCtrl: Generate worker config
    WorkerCtrl->>K8sAPI: Create TalosMachine CRs (Workers)
    
    K8sAPI->>MachineCtrl: Watch events
    MachineCtrl->>TalosAPI: Apply configuration
    TalosAPI->>Machines: Configure & bootstrap
    Machines-->>MachineCtrl: Status updates
    MachineCtrl->>K8sAPI: Update Machine status
    
    CPCtrl->>TalosAPI: Bootstrap etcd
    CPCtrl->>TalosAPI: Wait for cluster ready
    CPCtrl->>CPCtrl: Generate kubeconfig
    CPCtrl->>K8sAPI: Store kubeconfig in Secret
    CPCtrl->>K8sAPI: Update ControlPlane status
    
    WorkerCtrl->>TalosAPI: Join workers to cluster
    WorkerCtrl->>K8sAPI: Update Worker status
    
    ClusterCtrl->>K8sAPI: Update Cluster status (Ready)
    K8sAPI-->>User: Cluster ready!
```

## Module Interaction

```mermaid
graph LR
    subgraph "api/v1alpha1"
        CRDs[Custom Resource<br/>Definitions]
    end
    
    subgraph "internal/controller"
        Controllers[Controllers<br/>Reconciliation Logic]
    end
    
    subgraph "pkg"
        Talos[pkg/talos<br/>Talos Operations]
        Helm[pkg/helm<br/>Addon Management]
        Storage[pkg/storage<br/>Backup Storage]
        Utils[pkg/utils<br/>Utilities]
    end
    
    subgraph "cmd"
        Main[main.go<br/>Entry Point]
    end
    
    CRDs -->|Defines| Controllers
    Main -->|Starts| Controllers
    Controllers -->|Uses| Talos
    Controllers -->|Uses| Helm
    Controllers -->|Uses| Storage
    Controllers -->|Uses| Utils
    
    style CRDs fill:#e1f5ff
    style Controllers fill:#fff4e1
    style Talos fill:#ffe1e1
    style Helm fill:#ffe1e1
    style Storage fill:#ffe1e1
    style Utils fill:#ffe1e1
    style Main fill:#e1ffe1
```

## Deployment Modes

```mermaid
graph TB
    subgraph "Container Mode"
        TalosOp1[Talos Operator]
        K8sHost1[Host Kubernetes Cluster]
        TalosPods[Talos Pods<br/>Kubernetes-in-Kubernetes]
        GuestK8s1[Guest K8s Cluster<br/>inside Talos Pods]
        
        TalosOp1 -->|Creates| TalosPods
        K8sHost1 -->|Runs| TalosPods
        TalosPods -->|Runs| GuestK8s1
    end
    
    subgraph "Metal Mode"
        TalosOp2[Talos Operator]
        K8sHost2[Host Kubernetes Cluster]
        BareMetal[Physical/Virtual<br/>Machines]
        GuestK8s2[Guest K8s Cluster<br/>on Machines]
        
        TalosOp2 -->|Configures| BareMetal
        K8sHost2 -->|Runs Operator| TalosOp2
        BareMetal -->|Runs| GuestK8s2
    end
    
    style TalosPods fill:#e3f2fd
    style BareMetal fill:#fff3e0
    style GuestK8s1 fill:#f3e5f5
    style GuestK8s2 fill:#f3e5f5
```

## Controller Reconciliation Pattern

```mermaid
graph TD
    Start[Reconcile Request]
    Start --> Fetch[Fetch Resource]
    Fetch --> Exists{Resource<br/>Exists?}
    Exists -->|No| Delete[Handle Deletion]
    Exists -->|Yes| Analyze[Analyze Current State]
    
    Analyze --> Compare{Desired == Actual?}
    Compare -->|Yes| UpdateStatus[Update Status]
    Compare -->|No| Reconcile[Make Changes]
    
    Reconcile --> CreateRes[Create Child Resources]
    CreateRes --> ApplyConfig[Apply Configurations]
    ApplyConfig --> WaitReady[Wait for Ready]
    WaitReady --> UpdateStatus
    
    UpdateStatus --> Requeue{Need<br/>Requeue?}
    Requeue -->|Yes| Schedule[Schedule Next Run]
    Requeue -->|No| Done[Done]
    
    Delete --> Cleanup[Cleanup Resources]
    Cleanup --> RemoveFinalizer[Remove Finalizer]
    RemoveFinalizer --> Done
    
    style Start fill:#e8f5e9
    style Done fill:#e8f5e9
    style Compare fill:#fff3e0
    style Exists fill:#fff3e0
    style Requeue fill:#fff3e0
```

## Backup Flow

```mermaid
sequenceDiagram
    participant User
    participant BackupCtrl as Backup Controller
    participant TalosAPI as Talos API
    participant Etcd as etcd Cluster
    participant S3 as S3 Storage
    
    User->>BackupCtrl: Create TalosEtcdBackup CR
    BackupCtrl->>TalosAPI: Connect to control plane
    TalosAPI->>Etcd: Request snapshot
    Etcd-->>TalosAPI: Return snapshot data
    TalosAPI-->>BackupCtrl: Snapshot data
    BackupCtrl->>S3: Upload snapshot
    S3-->>BackupCtrl: Upload complete
    BackupCtrl->>BackupCtrl: Update backup status
    BackupCtrl-->>User: Backup complete!
```

## Addon Installation Flow

```mermaid
sequenceDiagram
    participant User
    participant AddonCtrl as Addon Controller
    participant K8sAPI as Kubernetes API
    participant HelmClient as Helm Client
    participant TargetCluster as Target Cluster
    
    User->>K8sAPI: Create TalosClusterAddon CR
    K8sAPI->>AddonCtrl: Watch event
    AddonCtrl->>K8sAPI: Fetch kubeconfig Secret
    K8sAPI-->>AddonCtrl: Kubeconfig
    AddonCtrl->>HelmClient: Initialize with kubeconfig
    HelmClient->>TargetCluster: Install/Upgrade Helm chart
    TargetCluster-->>HelmClient: Installation status
    HelmClient-->>AddonCtrl: Release info
    AddonCtrl->>K8sAPI: Create TalosClusterAddonRelease CR
    AddonCtrl->>K8sAPI: Update addon status
    K8sAPI-->>User: Addon installed!
```

## Package Dependencies

```mermaid
graph TD
    subgraph "Controllers depend on Packages"
        CPController[TalosControlPlane<br/>Controller]
        WorkerController[TalosWorker<br/>Controller]
        MachineController[TalosMachine<br/>Controller]
        BackupController[Backup<br/>Controller]
        AddonController[Addon<br/>Controller]
    end
    
    subgraph "Core Packages"
        TalosClient[talos/client.go]
        TalosBundle[talos/bundle.go]
        TalosCP[talos/control_plane.go]
        TalosWorker[talos/worker.go]
        TalosEtcd[talos/etcd.go]
        HelmClient[helm/client.go]
        S3Storage[storage/s3.go]
    end
    
    CPController --> TalosClient
    CPController --> TalosBundle
    CPController --> TalosCP
    
    WorkerController --> TalosClient
    WorkerController --> TalosWorker
    
    MachineController --> TalosClient
    
    BackupController --> TalosClient
    BackupController --> TalosEtcd
    BackupController --> S3Storage
    
    AddonController --> HelmClient
    
    style CPController fill:#bbdefb
    style WorkerController fill:#c5e1a5
    style MachineController fill:#fff9c4
    style BackupController fill:#ffccbc
    style AddonController fill:#f8bbd0
```

## Key Takeaways

1. **Layered Architecture**: Clear separation between API definitions, business logic (controllers), and utilities (packages)

2. **Controller Pattern**: Each resource type has a dedicated controller that continuously reconciles desired vs actual state

3. **Package Reusability**: Common operations are extracted into packages that multiple controllers can use

4. **Mode Flexibility**: The same operator can manage clusters in different environments (container vs metal mode)

5. **Kubernetes-Native**: Leverages Kubernetes API for state management, watches, and reconciliation

For detailed explanations, see:
- [Quick Reference](quick-reference.md) for quick lookup
- [Module by Module Guide](module-by-module-guide.md) for comprehensive details

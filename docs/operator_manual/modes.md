# Modes

Talos operator currently support two modes to operate: `container` and `metal`. The mode can be set any Custom Resource (CR) by setting the `spec.mode` field. It's a required field for all CRs. There is no default value for the mode, so you must set it explicitly.


## Container Mode

In `container` mode, the Talos operator will run Talos Linux in a containerized environment. This mode is useful for testing and development purposes, where you want to run Talos Linux without needing dedicated hardware or virtual machines. The operator will manage the lifecycle of the Talos Linux instances running in containers.

TODO: Put the diagram here

## Metal Mode

In `metal` mode, the Talos operator will manage Talos Linux instances running on bare metal or virtual machines. This mode is suitable for production environments where you want to run Talos Linux on dedicated hardware or virtual machines. The operator will handle the lifecycle of the Talos Linux instances, including provisioning, upgrading, and managing the configuration. You have to make sure that your machines are waiting at `Maintenance` mode before the operator can manage them. The operator requires the machines to be in `Maintenance` mode to perform operations like provisioning, upgrading, and managing the configuration. It can't take over machines that are already bootsrapped and running Talos Linux. If you want to delete your existing machines and recreate them by the operator, you can run `talosctl reset` command on the machines to put them in `Maintenance` mode. After that, the operator will be able to manage the machines.

The overview of the `metal` mode is as follows:


                +-----------------+
                |  TalosCluster   |
                +-----------------+
                   |           |
            owns   |           |    owns
                   v           v
     +----------------------+   +----------------------+
     | TalosControlPlane    |   |     TalosWorker      |
     |         CRD          |   |         CRD          |
     +----------------------+   +----------------------+
             |  mode=metal                |  mode=metal
             v                            v
     +----------------------+   +----------------------+
     | TalosMachine CRD     |   | TalosMachine CRD     |
     | (control-plane node) |   |   (worker node)      |
     +----------------------+   +----------------------+
             ^                            ^
             | ownerRef                   | ownerRef
             +------------+---------------+
                          |
         each machine CR is owned by its parent CRD -- (TalosControlPlane||TalosWorker)




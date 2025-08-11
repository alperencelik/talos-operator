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
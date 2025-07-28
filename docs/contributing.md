# Contributing

Thank you for your interest in contributing to the talos-operator!

You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster. The project is using [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) to generate the controller and CRDs. The controllers are located under `internal/controllers/` directory and the external packages such as `talos` is located under `pkg/` directory.

- To create a new controller you can use the following command:

```bash
kubebuilder create api --group talos --version v1alpha1 --kind TalosMachine
```

- Define the spec and status of your new kind in `api/v1alpha1/newkind_types.go` file.

- Define the controller logic in `internal/controllers/newkind_controller.go` file.

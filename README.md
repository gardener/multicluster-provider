# gardener/multicluster-provider

This repository contains an **experimental** provider implementation for [multicluster-runtime](https://github.com/multicluster-runtime/multicluster-runtime), a new [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) "addon" that allows writing uniform multi-cluster-aware Kubernetes controllers.

## Provider Flavors

The Gardener provider facilitates interaction with Gardener-managed clusters by watching specific resources and managing short-lived, auto-renewed kubeconfigs for secure access.
It supports two operational modes:

### `garden` Flavor

- **Functionality**: The controller communicates with the **garden cluster** and monitors `core.gardener.cloud/v1beta1.Shoot` resources.
- **Authentication**: Requests temporary [**admin kubeconfigs**](https://gardener.cloud/docs/gardener/shoot/shoot_access/) that are short-lived and automatically renewed for secure access to Shoot clusters.
- **Use Case**: Ideal for managing Shoot resources directly within the garden cluster.

### `seed` Flavor

- **Functionality**: The controller connects to a **seed cluster** and monitors `extensions.gardener.cloud/v1alpha1.Cluster` resources.
- **Authentication**: Utilizes the standard **cluster-admin kubeconfig** provided by the `gardenlet`, which is also short-lived and auto-renewed.
- **Use Case**: Suitable for managing Cluster resources within a seed cluster environment.

## Examples

See [examples/gardener](./examples/gardener) for sample code.

## Getting Started

To use the Gardener provider, ensure you have a running Gardener setup and the necessary permissions to access garden and/or seed clusters.
Detailed setup and configuration instructions can be found [here](https://gardener.cloud/docs/gardener/deployment/getting_started_locally/).

## Contributing

Thanks for taking the time to start contributing!

### Before you start

* Please familiarize yourself with the [Code of Conduct](./CODE_OF_CONDUCT.md) before contributing.
* See [CONTRIBUTING.md](./CONTRIBUTING.md) for instructions on the developer certificate of origin that we require.

### Pull requests

* We welcome pull requests. Feel free to dig through existing [issues](https://github.com/gardener/multicluster-provider/issues) and jump in.

## License

This project is licensed under [Apache-2.0](./LICENSE).

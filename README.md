# Ipruler Operator

## Overview

The Ipruler operator offers several Custom Resources (CRs) to efficiently manage the routing policies of your Kubernetes worker nodes. It utilizes [ipruler-agents](https://github.com/plutocholia/ipruler-agent) to configure Linux-based Kubernetes worker nodes, enabling logical grouping of nodes through the `NodeConfig` CR and defining cluster-wide policies using the `ClusterConfig` CR.

## The Way It Works

After the [installation](#installation), a DaemonSet for the [ipruler-agent](https://github.com/plutocholia/ipruler-agent) will be deployed, along with a single `ipruler-operator` deployment running a single replica.

To start injecting routing configurations, there must be a single `ClusterConfig` and at least one `NodeConfig` in the cluster. The operator will then create a third Custom Resource (CR) called `FullConfig`, named after the corresponding `NodeConfig`. The `FullConfig` CR contains a merged configuration derived from both the `ClusterConfig` and the `NodeConfig`. Once these configurations are merged, the `FullConfig` will inject its settings into the corresponding [ipruler-agents](https://github.com/plutocholia/ipruler-agent) based on the `NodeConfig`'s `spec.nodeSelector`.

## Examples

- [source-based-routing](./config/samples/custom/vlan-source-based-routing/manifests.yaml) sample.

## Limitations

- The [ipruler-agents](https://github.com/plutocholia/ipruler-agent) utilize the `Linux` `netlink` interface to create VLANs, routes, and rules. As a result, this operator, which is based on these agents, is currently limited to `Linux` and cannot be used on other operating systems.

- There must be only one `ClusterConfig` in the entire cluster. Currently, I'm working on implementing a validation webhook to prevent the creation of more than one `ClusterConfig` in the cluster, but it's not finished yet.

- There must be a single `ClusterConfig` (even an empty one) in the entire cluster to enable NodeConfigs to be injected into the agents.

- The `spec.nodeSelector` field in `NodeConfig` CR is immutable. To change the set of nodes associated with a `NodeConfig`, you need to delete the existing `NodeConfig` and create a new one. For more control over this process and to achieve your desired outcome, be sure to review the [cleanup policy](#cleaup-policy).

## Cleaup Policy

- By default, deleting a `NodeConfig` will result in the removal of the configuration from the corresponding nodes. You can disable this behavior in the `ipruler-operator` values file by setting `config.node-cleanup-on-deletion=false`.

## Installation

### Helm 

```bash
helm --namespace ipruler-operator upgrade --install \
    --create-namespace --repo https://plutocholia.github.io/ipruler-operator \
    ipruler-operator ipruler-operator --version x.x.x
```

### Default values

| Key                               | Description                      | Default                          |
|--------------------------------   |----------------------------------|----------------------------------|
| `image.repository`                | Docker image repository          | `plutocholia/ipruler-operator` |
| `image.tag`                       | Tag for the image                | `~` |
| `image.pullPolicy`                | Image pull policy                | `IfNotPresent` |
| `config.agent-api-port`           | Communication port to the ipruler-agent API | `9301` |
| `config.node-cleanup-on-deletion` | Whether to cleanup routing configurations on worker nodes on deletion of NodeConfigs | `true`|
| `resources.limits.cpu`            | CPU limits for the container | `500m` |
| `resources.limits.memory`         | Memory limits for the container | `128Mi` |
| `resources.requests.cpu`          | CPU requests for the container | `10m` |
| `resources.requests.memory`       | Memory requests for the container | `64Mi` |
| `replicas`                        | Number of operator replicas | `1` |
| `ipruler-agent.enabled`           | Enable installation of ipruler agent | `true` |
| `crds.enabled`                    | Enable installation of Custom Resource Definitions (CRDs) | `true` |

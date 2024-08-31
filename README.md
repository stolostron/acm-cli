# acm-cli

Collection of binaries for managing Red Hat Advanced Cluster Management for Kubernetes (RHACM),
delivered through a container image containing a Golang web server for the files to be readily
downloadable:

- **Policy toolset**
  - A toolset to help manage RHACM policies.
  - Code repository: https://github.com/stolostron/policy-cli
- **Policy generator**
  - Kustomize plugin binary to generate RHACM policies from Kubernetes manifests.
  - Code repository: https://github.com/stolostron/policy-generator-plugin
  - Files available:
    ```
    PolicyGenerator
    darwin-amd64-PolicyGenerator
    linux-amd64-PolicyGenerator
    windows-amd64-PolicyGenerator.exe
    ```
    **NOTE:** The `PolicyGenerator` binary matches the container architecture and is intended to be
    loaded into other containers like Openshift GitOps/ArgoCD. The files with architectures are
    served for users to be able to choose the binary that matches their local system.

## Build the image locally

```shell
make build-image
```

## Deploying the image

To deploy to any Kubernetes cluster, serving over a `NodePort` over HTTP:

```shell
make deploy
```

To deploy to an Openshift cluster with end-to-end encryption and Openshift manifests including a
`ConsoleCLIDownload` to display downloads in the console:

```shell
make deploy-openshift
```

## Testing the image

The `e2e-test` target builds the image, deploys to a Kind cluster, and verifies the files being
served against [`test/cli_list.html`](test/cli_list.html):

```shell
make e2e-test
```

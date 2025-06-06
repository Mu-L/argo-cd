# Tools

## Production

Argo CD supports several different ways in which Kubernetes manifests can be defined:

* [Kustomize](kustomize.md) applications
* [Helm](helm.md) charts
* [OCI](oci.md) images
* A directory of YAML, JSON, or [Jsonnet](jsonnet.md) manifests.
* Any [custom config management tool](../operator-manual/config-management-plugins.md) configured as a config management plugin

## Development
Argo CD also supports uploading local manifests directly. Since this is an anti-pattern of the
GitOps paradigm, this should only be done for development purposes. A user with an `override` permission is required
to upload manifests locally (typically an admin). All the different Kubernetes deployment tools mentioned above are supported.
To upload a local application:

```bash
$ argocd app sync APPNAME --local /path/to/dir/
```

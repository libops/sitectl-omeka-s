# sitectl-omeka-s

`sitectl-omeka-s` simplifies the creation and operation of repositories created using the [LibOps Omeka S template](https://github.com/libops/omeka-s). It provides sitectl commands for the Omeka S API, resource shortcuts, module maintenance, validation, and health checks.

Documentation: https://sitectl.libops.io/plugins/omeka-s

## Requirements

- Stable [`sitectl`](https://sitectl.libops.io/install) v1.0.0 or newer; this plugin uses RPC protocol 1.
- Docker with the Compose v2 plugin for local Omeka S sites.
- No additional app-plugin dependency beyond core `sitectl`.

## Quick Start

Create a local Omeka S site from the matching template:

```bash
sitectl create omeka-s/default \
  --template-repo https://github.com/libops/omeka-s \
  --path ./my-omeka-s-site \
  --type local \
  --checkout-source template \
  --default-context
```

The template README is at https://github.com/libops/omeka-s.

## Basic Operations

Use [`sitectl compose`](https://sitectl.libops.io/commands/compose) to start or inspect the stack:

```bash
sitectl compose up --remove-orphans -d
```

Use [`sitectl healthcheck`](https://sitectl.libops.io/commands/healthcheck) and [`sitectl validate`](https://sitectl.libops.io/commands/validate) to check the site:

```bash
sitectl healthcheck
sitectl validate
```

Use [`sitectl image`](https://sitectl.libops.io/commands/image) for local image or build-arg overrides:

```bash
sitectl image set --tag omeka-s=4.2.1-php84
```

The plugin intentionally does not register broad development bind mounts: mounting all modules or themes would hide extensions bundled in the versioned base image. Add custom extensions through the downstream build or an explicit per-extension override.

Use [`sitectl set`](https://sitectl.libops.io/commands/set) for component changes; it updates component-owned files immediately:

```bash
sitectl set ingress enabled --mode https-custom --domain omeka-s.localhost
sitectl set ingress enabled --trusted-ip 203.0.113.10/32 --max-upload-size 2G --upload-timeout 10m
```

See the [Omeka S plugin docs](https://sitectl.libops.io/plugins/omeka-s) for lifecycle operations, API helpers, resource shortcuts, and module maintenance.

## License

`sitectl-omeka-s` is licensed under the MIT License.

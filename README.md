# sitectl-omeka-s

`sitectl-omeka-s` simplifies the creation and operation of repositories created using the [LibOps Omeka S template](https://github.com/libops/omeka-s). It provides sitectl commands for the Omeka S API, resource shortcuts, module maintenance, validation, and health checks.

Documentation: https://sitectl.libops.io/plugins/omeka-s

## Requirements

- [`sitectl`](https://sitectl.libops.io/install).
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
sitectl image set --tag omeka-s=nginx-1.30.3-php84
```

Use [`sitectl set`](https://sitectl.libops.io/commands/set) and [`sitectl converge`](https://sitectl.libops.io/commands/converge) for component changes:

```bash
sitectl set ingress enabled --mode https-default --domain omeka-s.localhost
sitectl set ingress enabled --trusted-ip 203.0.113.10/32 --max-upload-size 2G --upload-timeout 10m
sitectl converge
```

See the [Omeka S plugin docs](https://sitectl.libops.io/plugins/omeka-s) for lifecycle operations, API helpers, resource shortcuts, and module maintenance.

## License

`sitectl-omeka-s` is licensed under the MIT License.

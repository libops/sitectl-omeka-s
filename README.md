# sitectl-omeka-s

`sitectl-omeka-s` simplifies the creation and operation of repositories created using the [LibOps Omeka S template](https://github.com/libops/omeka-s). It provides sitectl commands for the Omeka S API, resource shortcuts, module maintenance, validation, and health checks.

Documentation: https://sitectl.libops.io/plugins/omeka-s

## Requirements

- [`sitectl`](https://sitectl.libops.io/install) v1.8.2 or newer provides strict verification and deploy lifecycle hooks.
- Docker with the Compose v2 plugin for local Omeka S sites.
- No additional app-plugin dependency beyond core `sitectl`.
- Omeka S template v1.2.0 or newer provides the versioned runtime programs required by `sitectl-omeka-s` v1.3.0 and newer.

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
sitectl verify --strict
```

## Behavioral verification

`sitectl verify --strict` checks the running Omeka S version, scoped MariaDB identity, current `/admin` migration redirect, sites API collection shape, and files-volume access. The database probe reads the connection selected by the rendered `config/database.ini`; its password stays inside the container and is never copied into Docker process arguments or verifier output. Production verification is read-only.

Use `sitectl deploy` for application updates. Before stopping the current containers, the plugin checks that the site contains the matching template's rollout and verification programs and mounts each one read-only at its stable container path. A checkout older than template v1.2.0 fails with migration guidance before the outage; there is no inline fallback whose behavior could differ from the reviewed checkout.

Disposable CI may add a reversible service-account file write/read/delete probe:

```bash
sitectl verify --strict --disposable
```

Never use `--disposable` for a retained customer site. The checked-in migration gate keeps public Traefik stopped until the supported browser migration is complete. Hosted acceptance must still exercise a real prior-version database/files fixture, browser migration, public DNS/TLS, mail delivery, admin login, authenticated API mutation, and media retrieval.

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

# sitectl-omeka-s

`sitectl-omeka-s` is the LibOps sitectl plugin for Omeka S.

It registers a first-class create definition for `https://github.com/libops/omeka-s` so the stack can be installed with:

```bash
sitectl create omeka-s
```

It also provides context-aware helpers:

- `sitectl omeka-s build`
- `sitectl omeka-s init`
- `sitectl omeka-s up`
- `sitectl omeka-s down`
- `sitectl omeka-s status`
- `sitectl omeka-s logs [SERVICE...]`
- `sitectl omeka-s rollout`

Omeka S-specific helpers:

- `sitectl omeka-s api get RESOURCE [ID]`
- `sitectl omeka-s api request METHOD PATH`
- `sitectl omeka-s items [ID]`
- `sitectl omeka-s item-sets [ID]`
- `sitectl omeka-s media [ID]`
- `sitectl omeka-s vocabularies [ID]`
- `sitectl omeka-s resource-classes [ID]`
- `sitectl omeka-s properties [ID]`
- `sitectl omeka-s resource-templates [ID]`
- `sitectl omeka-s sites [ID]`
- `sitectl omeka-s site-pages [ID]`
- `sitectl omeka-s modules [ID]`

API helpers accept `--url`, `--identity`, `--credential`, and repeated `--query name=value` flags.

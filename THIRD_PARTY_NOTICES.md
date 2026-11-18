# Third-party notices

apibconv includes third-party software and embeds specification schema files.

## Embedded validation schemas

These files are embedded in the apibconv binary:

- `internal/specschema/schemas/openapi-2.0.json`
- `internal/specschema/schemas/openapi-3.0.json`
- `internal/specschema/schemas/openapi-3.1.json`

Source: OpenAPI Initiative, OpenAPI Specification JSON Schemas.
Copyright 2017 The Linux Foundation (OpenAPI 2.0 schema's historical source).
Copyright The Linux Foundation (OpenAPI Specification).

- https://spec.openapis.org/oas/2.0/schema/2017-08-27
- https://spec.openapis.org/oas/3.0/schema/2024-10-18
- https://spec.openapis.org/oas/3.1/schema/2025-02-13

These schemas are licensed under the Apache License, Version 2.0. A copy of
the license is included in [licenses/APACHE-2.0](licenses/APACHE-2.0). The
upstream license is available at
https://github.com/OAI/OpenAPI-Specification/blob/main/LICENSE.

## Software included in binaries

- **go.yaml.in/yaml/v3**: portions are licensed under MIT and the
  remaining files under Apache-2.0. See [licenses/GO-YAML](licenses/GO-YAML),
  [licenses/GO-YAML-NOTICE](licenses/GO-YAML-NOTICE), and
  [licenses/APACHE-2.0](licenses/APACHE-2.0).
- **github.com/santhosh-tekuri/jsonschema/v6**: Apache-2.0. See
  [licenses/APACHE-2.0](licenses/APACHE-2.0).
- **golang.org/x/text** and the Go standard library, including its
  vendored `golang.org/x/crypto`, `golang.org/x/net`, and `golang.org/x/text`
  packages: BSD-3-Clause. See [licenses/GO-BSD-3-CLAUSE](licenses/GO-BSD-3-CLAUSE)
  and the accompanying [patent grant](licenses/GO-PATENTS).

Review these notices and license files when updating dependencies or the Go
toolchain.

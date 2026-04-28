# Third-party notices

apibconv embeds third-party specification schema files so validation can run
reliably without network access.

## Embedded validation schemas

These files are embedded in the apibconv binary:

- `internal/specschema/schemas/openapi-2.0.json`
- `internal/specschema/schemas/openapi-3.0.json`
- `internal/specschema/schemas/openapi-3.1.json`

Source: OpenAPI Initiative, OpenAPI Specification JSON Schemas

- https://spec.openapis.org/oas/2.0/schema/2017-08-27
- https://spec.openapis.org/oas/3.0/schema/2024-10-18
- https://spec.openapis.org/oas/3.1/schema/2025-02-13
- License: Apache-2.0, https://github.com/OAI/OpenAPI-Specification/blob/main/LICENSE

These files are embedded in the apibconv binary:

- `internal/specschema/schemas/asyncapi-2.6.0.json`
- `internal/specschema/schemas/asyncapi-3.0.0.json`

Source: AsyncAPI Initiative, AsyncAPI specification JSON Schemas

- https://raw.githubusercontent.com/asyncapi/spec-json-schemas/master/schemas/2.6.0-without-$id.json
- https://raw.githubusercontent.com/asyncapi/spec-json-schemas/master/schemas/3.0.0-without-$id.json
- License: Apache-2.0, https://github.com/asyncapi/spec-json-schemas/blob/master/LICENSE

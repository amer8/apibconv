# Go integration

Use the Go packages to integrate conversion into your own application. The examples
show how to register parsers and writers and configure a converter. In a source
checkout, run them from the repository root with `go run ./examples/basic`, `go run ./examples/advanced`,
or `go run ./examples/plugin`. The first two create or replace `petstore.json` and
`transformed.yaml` in that directory.

The Go options support model validation, warning handling, and preservation of
OpenAPI `x-` extensions on the document and supported model objects, including
metadata, servers, paths, operations, parameters, request and response bodies,
media types, headers, schemas, components, and security schemes and flows.
Fields and extension locations that the shared model does not represent can still
be lost during conversion.

The built-in readers accept up to 16 MiB per input by default.
`converter.WithMaxInputBytes(ctx, n)` sets a positive byte limit for detection,
parsing and validation. Oversize input returns `errors.ErrInputTooLarge` from
the apibconv errors package. YAML processing also limits nesting to 128,
mappings to 2,048 entries, and total expanded nodes to 250,000. Aggregate
comparison and alias-expansion budgets can reject a document within those
individual limits. OpenAPI callback expansion shares a 4,096-visit budget and
a 32 MiB model-work budget across a document. Swagger server URL expansion
has a separate 32 MiB budget for the root and operation servers combined,
including when writing an OpenAPI 3 model as Swagger 2.0.
Each occurrence counts its URL length in bytes plus 128 bytes for the server entry.
With an implicit host, the scheme length takes the place of the URL length.

`converter.WithStrict(true)` validates the model before and after transformation
and treats warnings as errors. `converter.WithWarningHandler` enables the same
model checks and reports warnings at each stage; for example, an empty OpenAPI
`info.version` produces a warning. These model checks are not a substitute for
validating the serialized document against its specification schema.

`openapi.WithVersion` requires a matching input version: a major/minor version
accepts any patch, while a full version must match exactly. `openapi.WithStrict`
validates the input against the bundled schema before parsing. For API Blueprint,
`WithMarkdownParsing(true)` captures API, resource, and action descriptions;
the writer's `WithFormat` accepts the supported version, `1A`.

`model.API.Validate` checks model structure, local schema references, security,
and nested schemas across components, paths, webhooks, and callbacks.
`model.SecurityScheme.Validate` checks required fields for the scheme type.
These checks do not resolve external references or replace version-specific
specification validation.
See the [Go reference](https://pkg.go.dev/github.com/amer8/apibconv/pkg/converter)
for the package APIs.

- Basic [example](https://github.com/amer8/apibconv/blob/main/examples/basic/main.go)
- Advanced [example](https://github.com/amer8/apibconv/blob/main/examples/advanced/main.go)
- Custom transformation [example](https://github.com/amer8/apibconv/blob/main/examples/plugin/main.go)

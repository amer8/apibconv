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

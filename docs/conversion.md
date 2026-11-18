# Conversion behavior

Conversion is not lossless, including same-version encoding conversion: it goes
through a shared model and can omit or change fields. In particular, the API
Blueprint parser handles a subset of the format; Swagger operation-level
`schemes` and OpenAPI referenced response headers are not preserved. Keep the
original files, validate the output, and review it against the source before using
it as an API contract. Unsupported schema keywords and some security combinations
are rejected. Blueprint URI-template query variables are converted to query
parameters. Reserved, label, path-segment and matrix expansions (`+`, `.`, `/`,
`;`) and variable modifiers cannot be expressed faithfully and are rejected.
Fragment expressions are omitted because fragments are not sent in HTTP requests.
Blueprint request headers named `Authorization`, `Accept`, or `Content-Type` are
rejected with an explanation because OpenAPI ignores them as header parameters;
define the intended security and media types explicitly.

OpenAPI parameter, response, and request-body references remain references when
re-encoding OpenAPI 3; Swagger 2 parameter and response references are retained
when re-encoding Swagger 2. Swagger 2 references are retained on upgrade when
their media types match the reusable component; otherwise they are inlined to
preserve the operation's media types. When a resolved reference is modified in
the model, the modified value is written inline; unchanged references remain
reusable. Swagger body and form components are held in
`Swagger2Parameters` because they do not map directly to OpenAPI 3 parameters.

The Go `ParseToModel` and `WriteFromModel` methods and individual parser/writer
packages are lower-level building blocks. They do not enforce conversion direction
because a standalone model does not record a trustworthy source format. Custom
plugin formats registered with `Converter` define their own conversion rules.

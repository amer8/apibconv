# CLI reference

See `apibconv --help` for the full list of options.

Use `apibconv --licenses` to print the embedded third-party notices and license
texts for the current build target.

Options can appear before or after the input file. Use `--` before a filename
that starts with `-`. Only one input file is accepted, and `--validate` cannot
be combined with output options. CLI conversion supports OpenAPI output only;
an `.apib` or `.md` output path requires an explicit `--to openapi` override.
Encoding and output-version values are checked before conversion, including
defaults supplied through environment variables.

Explicit options take precedence over environment variables, which take
precedence over built-in defaults. `APIBCONV_DEFAULT_ENCODING` overrides the
output filename extension; without either, OpenAPI output defaults to YAML.
`APIBCONV_OPENAPI_VERSION` overrides the default target version, 3.1.

`--validate` checks the input without converting. For OpenAPI, it checks the
document against the bundled schema and runs the built-in model rules; for API
Blueprint, it parses the supported subset and runs the model rules. Text
diagnostics include severity, path, and message. `--report json` writes an object
with `valid`, `issues` (each with `level`, `path`, and `message`), and an
optional `error` to stdout once validation is attempted; operational messages
go to stderr. Error-level issues fail validation, while warnings do not.

By default, the CLI validates the generated OpenAPI document using the same
schema and model checks before sending it to stdout or replacing the output
file. It buffers the output in memory. If validation fails, no output document
is emitted or installed. These checks do not guarantee that conversion is
lossless or require a non-empty `info.version`; review the generated contract,
including its API version. Blueprint input has no API version: the CLI warns when it is missing, and `--api-version` supplies it. Use `--validate-output=false` to disable output checks. An explicit `--validate-output` and `--validate` are mutually
exclusive; `--report` applies to `--validate` only.

Output files are replaced atomically. On macOS and Linux, replacing an existing
file is refused when its ACL or ownership cannot be preserved safely. The
existing destination must be readable for change detection; changes detected
during conversion abort replacement. The final check and rename are separate
filesystem operations, so concurrent writers must coordinate or use distinct
output paths. On other platforms, choose a new output path when ACL preservation is unavailable.

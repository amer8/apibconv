# Integration Guide

This guide shows how to integrate `apibconv` into your GitHub Actions workflows.

## Quick Start

### Option 1: Direct Installation in Your Workflow

Add this step to your existing GitHub Actions workflow:

```yaml
- name: Convert OpenAPI to API Blueprint
  run: |
    go install github.com/amer8/apibconv@latest
    apibconv -f openapi.json -o api-blueprint.apib
```

### Option 2: Use the Reusable Workflow

Add a new workflow file `.github/workflows/convert-api.yml` to your repository:

```yaml
name: Convert API Documentation

on:
  push:
    branches: [ main ]
    paths:
      - 'openapi.json'
      - 'api/**/*.json'

jobs:
  convert:
    uses: amer8/apibconv/.github/workflows/reusable-apibconv.yml@main
    with:
      openapi-file: 'openapi.json'
      output-file: 'docs/api-blueprint.apib'
      apibconv-version: 'latest'
      upload-artifact: true
```

## Complete Integration Examples

### Example 1: Convert and Commit Back

Automatically convert your OpenAPI spec and commit the result:

```yaml
name: Update API Blueprint

on:
  push:
    branches: [ main ]
    paths:
      - 'openapi.json'

permissions:
  contents: write

jobs:
  convert:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Install apibconv
        run: go install github.com/amer8/apibconv@latest

      - name: Convert OpenAPI to API Blueprint
        run: apibconv -f openapi.json -o docs/api.apib

      - name: Commit changes
        run: |
          git config user.name "GitHub Actions"
          git config user.email "actions@github.com"
          git add docs/api.apib
          git diff --staged --quiet || git commit -m "Update API Blueprint documentation"
          git push
```

### Example 2: Convert Multiple Files

Convert multiple OpenAPI specifications:

```yaml
name: Convert All API Specs

on:
  push:
    branches: [ main ]

jobs:
  convert:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        spec:
          - { input: 'api/v1/openapi.json', output: 'docs/api-v1.apib' }
          - { input: 'api/v2/openapi.json', output: 'docs/api-v2.apib' }
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Install apibconv
        run: go install github.com/amer8/apibconv@latest

      - name: Convert ${{ matrix.spec.input }}
        run: apibconv -f ${{ matrix.spec.input }} -o ${{ matrix.spec.output }}

      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: api-blueprints
          path: docs/*.apib
```

### Example 3: Validate and Convert

Validate OpenAPI spec before converting:

```yaml
name: Validate and Convert API

on:
  pull_request:
    paths:
      - 'openapi.json'

jobs:
  validate-and-convert:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Validate OpenAPI Specification
        uses: char0n/swagger-editor-validate@v1
        with:
          definition-file: openapi.json

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Install apibconv
        run: go install github.com/amer8/apibconv@latest

      - name: Convert to API Blueprint
        run: apibconv -f openapi.json -o api-blueprint.apib

      - name: Comment PR with preview
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');
            const blueprint = fs.readFileSync('api-blueprint.apib', 'utf8');
            const preview = blueprint.split('\n').slice(0, 50).join('\n');

            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: `## API Blueprint Preview\n\n\`\`\`\n${preview}\n...\n\`\`\`\n\nFull file available as workflow artifact.`
            });

      - name: Upload API Blueprint
        uses: actions/upload-artifact@v4
        with:
          name: api-blueprint
          path: api-blueprint.apib
```

### Example 4: Docker-based Integration

Use Docker for conversion without installing Go:

```yaml
name: Convert API (Docker)

on:
  push:
    branches: [ main ]

jobs:
  convert:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Convert using Docker
        run: |
          docker run --rm \
            -v $(pwd):/data \
            -w /data \
            ghcr.io/amer8/apibconv:latest \
            -f openapi.json -o api-blueprint.apib

      - name: Upload result
        uses: actions/upload-artifact@v4
        with:
          name: api-blueprint
          path: api-blueprint.apib
```

## Integration with Documentation Sites

### Publish to GitHub Pages

```yaml
name: Publish API Documentation

on:
  push:
    branches: [ main ]

permissions:
  contents: read
  pages: write
  id-token: write

jobs:
  convert-and-publish:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Install apibconv
        run: go install github.com/amer8/apibconv@latest

      - name: Convert to API Blueprint
        run: apibconv -f openapi.json -o docs/api.apib

      - name: Setup Pages
        uses: actions/configure-pages@v4

      - name: Upload artifact
        uses: actions/upload-pages-artifact@v3
        with:
          path: 'docs'

      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v4
```

## CI/CD Pipeline Integration

### Add to Existing Test Pipeline

```yaml
name: CI

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run tests
        run: make test

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run linter
        run: make lint

  convert-api-docs:
    runs-on: ubuntu-latest
    needs: [test, lint]  # Only convert if tests pass
    if: github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v4

      - name: Install apibconv
        run: go install github.com/amer8/apibconv@latest

      - name: Convert API documentation
        run: apibconv -f openapi.json -o docs/api.apib

      - name: Upload documentation
        uses: actions/upload-artifact@v4
        with:
          name: api-documentation
          path: docs/api.apib
```

## Performance Benchmarks

This tool is designed for zero allocations in buffer operations. The GitHub Actions CI includes benchmarks that verify this:

```yaml
- name: Verify zero allocations
  run: |
    go test -bench=BenchmarkWriteAPIBlueprint -benchmem ./converter
    # This will fail if allocations are detected
```

You can add similar checks to your pipeline:

```yaml
- name: Run conversion benchmarks
  run: |
    go install github.com/amer8/apibconv@latest
    go test -bench=. -benchmem github.com/amer8/apibconv/converter
```

## Troubleshooting

### Common Issues

1. **Go not installed**: Use the Docker image or install Go in your workflow
2. **Permission denied**: Ensure workflow has appropriate permissions for file operations
3. **File not found**: Check paths are relative to repository root

### Getting Help

- Check the [README](README.md) for basic usage
- Review [example workflows](.github/workflows/)
- Open an issue on GitHub if you encounter problems

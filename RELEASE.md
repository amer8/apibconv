# Release Process

This document describes how releases are created and published for apibconv.

## Automated Release Pipeline

Releases are fully automated via GitHub Actions when you push a git tag.

### Release Workflow

When you push a tag (e.g., `v0.4.0`), the following happens automatically:

1. **Tests Run** - All tests must pass
2. **Zero Allocation Check** - Benchmarks verify 0 allocs/op
3. **Multi-Platform Builds** - GoReleaser builds binaries for:
   - Linux (amd64, arm64, arm)
   - macOS (amd64, arm64)
   - Windows (amd64)
4. **Docker Image** - Multi-arch image built and pushed to ghcr.io:
   - `ghcr.io/amer8/apibconv:latest`
   - `ghcr.io/amer8/apibconv:v0.4.0`
   - `ghcr.io/amer8/apibconv:v0.4`
   - `ghcr.io/amer8/apibconv:v0`
5. **GitHub Release** - Created with:
   - Release notes (auto-generated from commits)
   - Binary archives for all platforms
   - Checksums
   - Changelog

## Creating a New Release

### 1. Update Version Documentation (Optional)

If you want to document the release, update:
- `CHANGELOG.md` (if you maintain one)
- Any version-specific documentation

### 2. Create and Push Tag

```bash
# Make sure you're on main/master and up to date
git checkout main
git pull

# Create an annotated tag
git tag -a v0.4.0 -m "Release v0.4.0"

# Push the tag to trigger release workflow
git push origin v0.4.0
```

### 3. Monitor Release

1. Go to GitHub Actions: `https://github.com/amer8/apibconv/actions`
2. Watch the "Release" workflow
3. If successful, check:
   - GitHub Releases: `https://github.com/amer8/apibconv/releases`
   - Docker image: `https://github.com/amer8/apibconv/pkgs/container/apibconv`

### 4. Verify Release

```bash
# Test Go installation
go install github.com/amer8/apibconv@v0.4.0
apibconv -version

# Test Docker image
docker pull ghcr.io/amer8/apibconv:v0.4.0
docker run --rm ghcr.io/amer8/apibconv:v0.4.0 -version

# Test binary download
wget https://github.com/amer8/apibconv/releases/download/v0.4.0/apibconv_0.4.0_Linux_x86_64.tar.gz
tar -xzf apibconv_0.4.0_Linux_x86_64.tar.gz
./apibconv -version
```

## Release Artifacts

Each release includes:

### Binaries
- `apibconv_VERSION_Linux_x86_64.tar.gz`
- `apibconv_VERSION_Linux_arm64.tar.gz`
- `apibconv_VERSION_Darwin_x86_64.tar.gz` (macOS Intel)
- `apibconv_VERSION_Darwin_arm64.tar.gz` (macOS Apple Silicon)
- `apibconv_VERSION_Windows_x86_64.zip`
- `checksums.txt`

### Docker Images
- Multi-architecture support (amd64, arm64)
- Available at: `ghcr.io/amer8/apibconv`
- Tags: `latest`, `vX.Y.Z`, `vX.Y`, `vX`

### Source Code
- Automatic source archives (`.tar.gz` and `.zip`)

## Version Numbering

We follow [Semantic Versioning](https://semver.org/):

- `v0.4.0` - Major.Minor.Patch
- `v0.4.0-rc.1` - Release candidate
- `v0.4.0-beta.1` - Beta release

### When to Bump

- **Major (v1.0.0)**: Breaking changes to CLI or API (Stable release)
- **Minor (v0.5.0)**: New features, backward compatible
- **Patch (v0.4.1)**: Bug fixes, backward compatible

## Changelog Generation

GoReleaser automatically generates changelogs from commit messages:

### Commit Message Format

Use conventional commits for better changelogs:

```bash
feat: add support for OpenAPI 3.1
fix: correct parameter parsing for optional fields
perf: improve JSON parsing by 42% with jsoniter
docs: update installation instructions
chore: update dependencies
```

These are grouped in the release notes:
- `feat:` → New Features
- `fix:` → Bug Fixes
- `perf:` → Performance Improvements

## Docker Image Details

### Image Structure

```dockerfile
FROM scratch
- Non-root user (uid 65532)
- Working directory: /data
- Binary at: /apibconv
```

### Usage

```bash
# Basic usage
docker run --rm -v $(pwd):/data ghcr.io/amer8/apibconv:latest openapi.json -o output.apib

# With specific version
docker run --rm -v $(pwd):/data ghcr.io/amer8/apibconv:v0.4.0 openapi.json -o output.apib
```

### Platforms Supported

- linux/amd64
- linux/arm64

## Troubleshooting Releases

### Release Workflow Fails

**Tests Fail:**
```bash
# Run tests locally first
go test ./...
```

**Zero Allocation Check Fails:**
```bash
# Verify zero allocations locally
go test -bench=BenchmarkWriteAPIBlueprint -benchmem ./converter
# Look for: 0 B/op  0 allocs/op
```

**GoReleaser Fails:**
```bash
# Test GoReleaser locally (requires GoReleaser installed)
goreleaser release --snapshot --clean
```

### Docker Build Fails

```bash
# Test Docker build locally
docker build -t apibconv:test .
docker run --rm apibconv:test -version
```

### Tag Already Exists

If you need to recreate a tag:

```bash
# Delete local tag
git tag -d v0.4.0

# Delete remote tag (careful!)
git push --delete origin v0.4.0

# Recreate and push
git tag -a v0.4.0 -m "Release v0.4.0"
git push origin v0.4.0
```

## Manual Release (Emergency)

If the automated workflow fails, you can create a manual release:

### Build Binaries

```bash
# Install GoReleaser
go install github.com/goreleaser/goreleaser@latest

# Create release
goreleaser release --clean
```

### Build Docker Image

```bash
# Build multi-arch image
docker buildx create --use
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t ghcr.io/amer8/apibconv:v0.4.0 \
  --push .
```

## Release Checklist

Before creating a release:

- [ ] All tests pass: `go test ./...`
- [ ] Benchmarks pass: `go test -bench=. ./...`
- [ ] Zero allocations verified
- [ ] Documentation updated
- [ ] Version number decided
- [ ] CHANGELOG updated (optional)
- [ ] Main branch is clean and up to date

After release:

- [ ] GitHub release created successfully
- [ ] Docker image pushed to ghcr.io
- [ ] Binaries available for download
- [ ] `go install` works with new version
- [ ] Docker pull works with new tag
- [ ] Release notes look correct

## Configuration Files

The release process uses these configuration files:

- `.github/workflows/release.yml` - GitHub Actions workflow
- `.goreleaser.yml` - GoReleaser configuration
- `Dockerfile` - Docker image definition
- `.dockerignore` - Docker build exclusions

## Security

### Image Scanning

Docker images are automatically scanned by GitHub's security features. Check:
`https://github.com/amer8/apibconv/security`

### Binary Checksums

All binaries include SHA256 checksums in `checksums.txt`:

```bash
# Verify a downloaded binary
sha256sum -c checksums.txt
```

### Signing (Future)

Consider adding in future releases:
- GPG signing of binaries
- Docker image signing with cosign
- SBOM (Software Bill of Materials)

## Support

For issues with releases:
1. Check GitHub Actions logs
2. Check existing issues: `https://github.com/amer8/apibconv/issues`
3. Create a new issue with `release` label

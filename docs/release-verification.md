# Release verification

- published release assets cover Linux and macOS on `x64` and `arm64`
- each platform archive ships with a matching `*_checksums.txt` file for targeted verification
- signed releases also ship a matching `*.sigstore.json` bundle for the checksum file
- each release also includes a CycloneDX `bom.json` SBOM

Older releases may only have the checksum file. In that case, verify the checksum directly.

For example, set the release tag and archive name for the file you downloaded.
Run the commands in the directory containing the archive, its checksum file,
and its signature bundle. On Linux x86-64:

```sh
tag=vX.Y.Z
archive=apibconv_X.Y.Z_Linux_x86_64.tar.gz
```

On macOS arm64:

```sh
tag=vX.Y.Z
archive=apibconv_X.Y.Z_Darwin_arm64.tar.gz
```

Replace `X.Y.Z` in both values with the release version, and use the matching
archive name for your architecture. Verify the signature on that archive's
checksum file:

```sh
cosign verify-blob \
  --bundle "${archive}_checksums.txt.sigstore.json" \
  --certificate-identity "https://github.com/amer8/apibconv/.github/workflows/release.yml@refs/tags/${tag}" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  "${archive}_checksums.txt"
```

Once the signature verification succeeds, verify the downloaded archive against
that same checksum file. Each checksum file contains only the SHA-256 digest,
so the commands below add the archive filename in the format expected by the
checksum tool. On Linux:

```sh
printf '%s  %s\n' "$(cat "${archive}_checksums.txt")" "$archive" | sha256sum -c -
```

On macOS:

```sh
printf '%s  %s\n' "$(cat "${archive}_checksums.txt")" "$archive" | shasum -a 256 -c -
```

Releases produced by the provenance-enabled release workflow additionally include
`apibconv_X.Y.Z_provenance.sigstore.json` and `apibconv_X.Y.Z_provenance.intoto.jsonl`.
The native Sigstore bundle contains signed SLSA build provenance covering every platform
archive, checksum file, checksum signature bundle, and the SBOM. The JSONL file exports
the same signed in-toto envelope for tools such as OpenSSF Scorecard.

For an immutable release, verify GitHub's release attestation and a downloaded archive:

```sh
gh release verify vX.Y.Z --repo amer8/apibconv
gh release verify-asset vX.Y.Z apibconv_X.Y.Z_Linux_x86_64.tar.gz --repo amer8/apibconv
```

Verify the archive's build provenance using its downloaded native bundle:

```sh
gh attestation verify apibconv_X.Y.Z_Linux_x86_64.tar.gz \
  --repo amer8/apibconv \
  --bundle apibconv_X.Y.Z_provenance.sigstore.json \
  --cert-identity "https://github.com/amer8/apibconv/.github/workflows/release.yml@refs/tags/vX.Y.Z" \
  --source-ref refs/tags/vX.Y.Z \
  --deny-self-hosted-runners
```

Add `--source-digest <expected-commit-sha>` to require a particular source commit.
Build provenance identifies the source and build workflow. GitHub's release attestation
binds the published tag, commit, and release assets. Older releases may lack either
attestation; use the checksum and signature checks available for those releases.

Container provenance is published with the image in GHCR. After authenticating to GHCR,
verify an image by its exact digest, replacing `IMAGE_DIGEST` with the SHA-256 digest:

```sh
gh attestation verify oci://ghcr.io/amer8/apibconv@sha256:IMAGE_DIGEST \
  --repo amer8/apibconv --bundle-from-oci \
  --cert-identity "https://github.com/amer8/apibconv/.github/workflows/release.yml@refs/tags/vX.Y.Z" \
  --source-ref refs/tags/vX.Y.Z \
  --deny-self-hosted-runners
```

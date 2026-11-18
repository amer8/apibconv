# Releasing

Enable **Settings → Releases → Enable release immutability** before publishing with
the provenance-enabled workflow. Immutability applies to future releases. Merge the
release workflow changes before creating the next version tag.

The workflow builds the standard-library-only Go release helper once per job.
Run `go run ./internal/releasetool --help` to inspect its commands locally.

The release workflow runs in this order:

1. Build and sign all platform archives once, upload them to a draft release, and
   smoke-test the same artifacts through the install action.
2. Generate build provenance, attach the verification bundle and signed in-toto
   envelope, then download and verify the complete draft asset inventory.
3. Publish the completed draft. GitHub freezes its assets and tag and generates
   the release attestation.
4. Download the published assets and verify their checksums, Cosign signatures,
   build provenance, and release attestation. Verification requires the expected
   workflow identity, tag, and source commit, and rejects a modified archive.
5. Build and push the container image, generate provenance for its exact digest,
   and verify the provenance from GHCR.

If **Build and Verify Draft** fails, leave the release as a draft and rerun the
failed jobs. The next attempt replaces the incomplete draft and rebuilds its assets.
If **Publish Completed Draft** loses its response after publishing, rerunning it
recognizes the existing release and continues to verification.

If **Verify Published Release** or the Docker job fails, rerun only the failed jobs
from the original workflow run. The build job refuses to rebuild a published version.
Published asset corrections require a new version tag. Container attestations can
be retried by rerunning the Docker job; consumers should pin and verify image digests.

GitHub release attestations are generated automatically after immutability is enabled.
OpenSSF Scorecard currently discovers build provenance through the attached
`*.intoto.jsonl` asset and evaluates the last five releases, so its Signed-Releases
score improves as releases with provenance enter that window.

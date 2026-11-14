# Independent fork releases

`backend/cmd/server/FORK_VERSION` is the only application version source for TrueRouter Sub2API. `backend/cmd/server/VERSION` records the upstream version and follows `Wei-Shaw/sub2api` unchanged. Upstream merges must not replace the fork version.

Read the mandatory policy in the repository root `AGENTS.md` before changing versions or workflows.

## Release sequence

1. Choose the next fork version independently of upstream.
2. Update `FORK_VERSION` and commit it with the release changes.
3. Run the local end-to-end check below. Review its JSON evidence.
4. After publication is authorized, create an annotated tag `tr-v<version>` on that exact commit. The tag and `FORK_VERSION` must match.
5. Push the authorized source and tag to `true-router/sub2api`.
6. The release workflow validates the tag, version file, and source commit. It also runs the fork-version E2E gate before publication.
7. Verify the resulting archive checksums, binary version, image version, and release source identity.

For example, the independent version `0.2.26` uses tag `tr-v0.2.26`, archive `sub2api_0.2.26_linux_amd64.tar.gz`, and image `ghcr.io/true-router/sub2api:0.2.26`. Existing `v*` tags are preserved for historical fork updates and rollback. New fork releases must not use them. Older source revisions without `FORK_VERSION` cannot be released by this workflow.

Release preparation never rewrites a version file. There is no post-release synchronization commit. Releasing an older tag cannot downgrade the default branch.

## Build and publication

The workflow builds the frontend once. GoReleaser OSS builds each selected Go target on its own runner in snapshot mode. The matrix comes from `.goreleaser.yaml`; simple mode selects Linux amd64 only. Snapshot packaging receives a synthetic `v<version>` tool identity and the numeric `RELEASE_VERSION`. It does not create or publish a Git tag.

Every archive has a source commit, target, numeric fork version, and SHA256 manifest. The publishing job verifies the complete matrix, extracts Linux binaries for image contexts, and prepares checksums and release notes. `gh release create --verify-tag` publishes the existing assets under the real `tr-v*` tag. This keeps the independent namespace compatible with GoReleaser OSS. Full releases contain the selected archives and checksums. Simple releases contain the amd64 GHCR image and release notes only.

Go caches remain isolated by target. Release runs remain serialized. Updates and installation use `true-router/sub2api`. Cached release information must include that repository identity; old upstream cache data is rejected.

## Local verification

```sh
python3 -m pip install -r .github/release-tools/requirements-release.txt
python3 .github/release-tools/check-fork-version-e2e.py --output .planning/fork-version-evidence/summary.json
bash -n deploy/install.sh
bash -n .github/release-tools/release-images.sh
```

The E2E check uses real isolated Git histories, release CLI commands, checksum verification, a real application build, and the real update HTTP handlers and GitHub repository client. Archive payloads, GitHub responses, and cache data are local fixtures. It checks independent upstream merges, tag rejection, full and simple packaging, corrupted archives, update lookup, cache migration, and historical rollback versions. Its receipt contains source hashes, a binary hash, and cleanup status. It neither publishes nor installs an update.

CI runs this check and uploads `fork-version-policy`. The release workflow independently uploads `release-fork-version-policy` and requires it before publication.

## Hosted dry run

After an authorized workflow dispatch, use a branch containing the new version policy:

```sh
gh workflow run release.yml --ref <branch> \
  -f tag=<branch> -f dry_run=true -f simple_release=false
```

A hosted dry run builds all selected archives and runtime images. It exports local OCI archives instead of pushing images. It prepares checksums, release notes, and metadata without GitHub publication, registry login, DockerHub description updates, or Telegram notification. Test simple mode separately with `simple_release=true`. Local E2E success does not prove a hosted build or production deployment.

## Deployment receiver migration

The independent `true-router/deployer` receiver must be updated before automatic rollout is enabled for new fork tags. It must accept stable `tr-vX.Y.Z` tags for `sub2api` only. Other components retain `vX.Y.Z`. Preserve the actual tag in signed notifications, queue identities, and receipts. Existing queued events must be handled before replacing the receiver; do not rewrite their signed identity.

# Repository instructions

## Mandatory independent fork release policy

- This repository publishes TrueRouter Sub2API from `true-router/sub2api`.
- The only fork application version source is `backend/cmd/server/FORK_VERSION`.
- Keep the fork version independent of the upstream release sequence. The migration retains `0.2.25`. Select subsequent versions from fork release history; never derive them from upstream.
- `backend/cmd/server/VERSION` belongs to upstream `Wei-Shaw/sub2api`. Accept upstream changes to this file during merges. Never write our release version to it.
- Every new release tag must be `tr-v<version>`, for example `tr-v0.2.26`. The tag must point to the commit containing the identical `FORK_VERSION` value.
- Never create, move, or publish `v*` tags for new fork releases. Preserve existing tags and historical fork releases.
- Displayed versions, binary versions, archive names, and container image tags use the numeric fork version without `tr-v`.
- Build entry points must read `FORK_VERSION`. The version resolver must ignore upstream tags and reject a conflicting exact fork tag.
- Release preparation must reject a wrong tag prefix, a version mismatch, or a tag pointing at another commit. Never rewrite either version file to make validation pass.
- Commit the chosen `FORK_VERSION` before creating a release tag. Do not synchronize version files back to the default branch after release. Releasing an older tag must never downgrade the branch.
- Update checks, rollback, and installation must use `true-router/sub2api`. Historical `v*` tags are readable only from this fork repository.
- Automatic rollout must retain the actual `tr-v*` tag in the signed notification. The independent deployment receiver must support this namespace before automatic rollout is enabled.
- GoReleaser OSS may receive a synthetic numeric `v<version>` identity only for local snapshot packaging with publication disabled. It must not create or publish that tag. Publish GitHub Releases with the validated real `tr-v*` tag.
- Preserve this policy and the release gates when merging upstream workflow changes. Do not restore upstream version synchronization or upstream update endpoints.

## Verification and change scope

- Use end-to-end checks. Do not add unit tests after implementation.
- Run `python3 .github/release-tools/check-fork-version-e2e.py --output <evidence-directory>/summary.json` after version-policy changes. Keep its verifiable JSON receipt outside tracked source.
- Run the fork-version E2E workflow before publishing a new release. Check the release tag, source commit, binary version, archive checksums, and image version together.
- Separate local verification, CI execution, publication, and deployment. Do not claim a later stage completed from evidence for an earlier stage.
- Do not push, publish, or deploy unless the user authorizes that action.
- Use Simplified Chinese in conversation. Write documentation in English.

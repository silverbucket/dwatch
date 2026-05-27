# Releasing dwatch

Releases are cut from GitHub Actions. Branch protection on `master` stays enabled.

Pre-built binaries: [GitHub releases](https://github.com/silverbucket/dwatch/releases).

## One-time setup: `release` environment

**Settings** → **Environments** → **New environment** → `release`.

Recommended:

- **Required reviewers** before publish
- **Deployment branches** → `master` only

**Settings** → **Actions** → **General** → workflow permissions **Read and write**.

## Cut a release

1. **Actions** → **Release** → **Run workflow** → **patch** (default), **minor**, or **major**
2. Approve the **release** environment if reviewers are configured
3. When the job finishes, merge the **Makefile bump PR** if one was opened (so `master` matches the release)

The workflow refuses to publish if that tag or release already exists.

## Version bumps

| Bump | Example (latest `v1.3.0`) |
|------|---------------------------|
| **patch** | `v1.3.1` |
| **minor** | `v1.4.0` |
| **major** | `v2.0.0` |

The starting point is the latest stable GitHub release tag (drafts and pre-releases excluded). If there are no releases yet, the Makefile `VERSION` on `master` is used.

## What the workflow does (order of operations)

1. **Compute** the new version from the latest release plus your bump choice.
2. **Edit the Makefile in the CI workspace** to that version (this happens **before** build; it does not commit to `master` yet).
3. **Run tests** on that tree.
4. **Build** binaries. The job checks that the Makefile matches the release version, then compiles with `-ldflags` so `dwatch --version` is correct.
5. **Publish** the GitHub Release and upload tarballs.
6. **Open a pull request** to update `VERSION` on `master`, because branch protection requires a PR for commits on `master`.

Step 6 does not change what is inside the tarballs from step 4. It only brings `master` in line for anyone cloning the repo and running `make build` locally.

# Releasing dwatch

Releases are cut from GitHub Actions on protected `master` — no local `git tag` or push required.

Pre-built binaries for end users are listed on the [GitHub releases](https://github.com/silverbucket/dwatch/releases) page.

## One-time setup: `release` environment

In the repo on GitHub: **Settings** → **Environments** → **New environment** → name it `release`.

Recommended for a protected `master` branch:

- **Required reviewers** — one or more people must approve before binaries are published
- **Deployment branches** — limit to `master` only (releases are built from `master` HEAD)

Ensure **Settings** → **Actions** → **General** → **Workflow permissions** is **Read and write**.

## Cut a release

1. **Actions** → **Release** → **Run workflow** — choose **patch** (default), **minor**, or **major**
2. CI reads the **latest published release** (e.g. `v1.3.0`), computes the next version (`1.3.1` for patch), runs tests, and publishes binaries
3. Approve the **release** environment deployment if reviewers are configured
4. Merge the automated **Makefile bump PR** when it appears (required for protected `master`; keeps `make build` in sync)

The workflow refuses to publish if that tag or GitHub Release already exists.

## Version bumps

| Bump | Example (`v1.3.0` latest) |
|------|-----------------------------|
| **patch** (default) | `v1.3.1` |
| **minor** | `v1.4.0` |
| **major** | `v2.0.0` |

Current version is taken from the latest GitHub release tag. If there are no releases yet, the workflow falls back to `VERSION` in the Makefile.

## Protected `master`

The release job does **not** push directly to `master`. After a successful publish, it opens a pull request to update `VERSION` in the Makefile. Merge that PR so local `make build` matches the release.

# dwatch

Track disk space growth over time. `dwatch` takes periodic snapshots of directory sizes and shows you what is growing — by bytes, percentage, or daily rate.

Works on macOS and Linux.

![dwatch demo](demo/demo.gif)


## Install

Requires Go 1.21+.

```sh
git clone <repo>
cd dwatch
make install
```

This builds the binary, installs it to `/usr/local/bin/dwatch`, and copies a default config to `~/.dwatch/dwatch.conf`.

```sh
make uninstall   # remove the binary
```

### Install from a release

Pre-built binaries are on the [GitHub releases](https://github.com/silverbucket/dwatch/releases) page (`linux` arm64/amd64, `darwin` arm64/amd64 for Apple Silicon and Intel Mac).

```sh
# Example: Linux arm64
curl -sL https://github.com/silverbucket/dwatch/releases/download/v1.3.0/dwatch_v1.3.0_linux_arm64.tar.gz | tar -xz
sudo install -m 755 dwatch /usr/local/bin/dwatch
```


## Quick start

```sh
dwatch scan                        # take a snapshot
dwatch status                      # largest dirs + what grew since last scan
dwatch diff --since 1w             # full change table for the past week
dwatch top --since 1m              # rank directories by growth this month
dwatch top --since 1d --by pct    # catch small dirs growing fast
```


## How it works

`dwatch scan` walks the filesystem, computes the allocated size of each directory using block counts (like `du` — sparse files and VM disk images report correctly), and saves a timestamped JSON snapshot to `~/.dwatch/`.

The reporting commands — `status`, `diff`, `top`, `alert` — compare any two snapshots. Snapshots are plain JSON files; prune them by deleting files directly.


## Cron setup

Run `scan` automatically on a schedule, then query on demand or alert via cron.

```
MAILTO=you@example.com

# Snapshot every 3 hours
0 */3 * * *   /usr/local/bin/dwatch scan --quiet

# Alert at 8am if anything grew >500 MB overnight
0 8 * * *     /usr/local/bin/dwatch alert --since 1d --threshold 500mb

# Weekly growth report every Monday
0 9 * * 1     /usr/local/bin/dwatch top --since 1w --limit 20
```

cron sends `MAILTO` an email when a command exits non-zero — so `alert` will mail you only when the threshold is exceeded.


## Commands

### `dwatch scan`

Walk the filesystem and save a snapshot.

| Flag | Default | Description |
|------|---------|-------------|
| `-r, --root` | `/` | Root directory to scan |
| `-n, --depth` | `5` | Max directory depth to track |
| `-s, --skip` | see config | Paths to exclude (repeatable) |
| `--show` | `10` | Largest dirs to print after scan (`0` = all) |
| `-q, --quiet` | `false` | One-line summary only; no table (for cron) |

Default skip paths — **macOS:** `/dev`, `/System/Volumes`, `/net`, `/home` · **Linux:** `/proc`, `/sys`, `/dev`, `/run`

```sh
dwatch scan --root /Users --depth 4
dwatch scan --skip /Volumes/Backup
dwatch scan --quiet
```


### `dwatch status`

Quick overview: largest directories in the latest snapshot and what grew.

| Flag | Default | Description |
|------|---------|-------------|
| `--since` | — | Compare over this window instead of since the previous scan |
| `-l, --limit` | `5` | Max entries per section (0 = all) |

```sh
dwatch status
dwatch status --since 1w
dwatch status --limit 10
```


### `dwatch diff`

Full change table between two snapshots.

| Flag | Default | Description |
|------|---------|-------------|
| `-s, --since` | — | Baseline time; omit to compare against the previous snapshot |
| `--min-change` | `1mb` | Minimum change to include |
| `-l, --limit` | `30` | Max rows (`0` = all) |
| `-a, --all` | `false` | Include unchanged directories |

```sh
dwatch diff --since 1w
dwatch diff --since 2026-05-01 --min-change 100mb
dwatch diff --all --limit 0
```


### `dwatch top`

Rank directories by growth within a time window.

| Flag | Default | Description |
|------|---------|-------------|
| `-s, --since` | — | Time window; omit to compare against the previous snapshot |
| `--by` | `growth` | Sort: `growth` (bytes), `pct` (percentage), `rate` (bytes/day) |
| `-l, --limit` | `20` | Number of results (`0` = all) |

`--by pct` is useful for catching directories that are small but growing fast — a cache that doubled in size ranks above a large directory that barely moved.

```sh
dwatch top --since 1w
dwatch top --since 1d --by pct
dwatch top --since 1m --by rate --limit 10
```


### `dwatch alert`

Exit 1 if any directory grew past a threshold. Silent on success — safe for scripts.

| Flag | Default | Description |
|------|---------|-------------|
| `-t, --threshold` | *(required)* | Growth threshold, e.g. `500mb`, `1gb` |
| `-s, --since` | — | Comparison window; omit to compare against the previous snapshot |

```sh
dwatch alert --threshold 1gb --since 1d
dwatch alert --threshold 500mb
```


## Global flags

| Flag | Default | Description |
|------|---------|-------------|
| `-d, --data-dir` | `~/.dwatch` | Where snapshots are stored |


## Configuration

`make install` copies a default config to `~/.dwatch/dwatch.conf`. Edit it to set your own defaults — CLI flags always take precedence.

```
data_dir  = ~/.dwatch
scan_root = /
scan_depth = 5
scan_skip = /dev, /System/Volumes, /net, /home
```

`scan_skip` is applied at report time, not just at scan time. Adding a path to the skip list removes it from `diff`, `top`, `status`, and `alert` output immediately — even for old snapshots — without needing to re-scan.

## Releasing

Releases are cut from CI on protected `master` — no local `git tag` or push required.

### One-time setup: `release` environment

In the repo on GitHub: **Settings** → **Environments** → **New environment** → name it `release`.

Recommended for a protected `master` branch:

- **Required reviewers** — one or more people must approve before binaries are published
- **Deployment branches** — limit to `master` only (tags/releases are built from `master` HEAD)

Ensure **Settings** → **Actions** → **General** → **Workflow permissions** is **Read and write**.

### Cut a release

1. Bump `VERSION` in the **Makefile** on `master` (e.g. `1.3.1`) and merge
2. **Actions** → **Release** → **Run workflow** — enter **major**, **minor**, and **patch** (integers) to match the Makefile (the job fails if they differ)
3. Approve the **release** environment deployment if reviewers are configured
4. CI checks out `master`, runs tests, builds archives, creates `v1.3.1`, and uploads assets to GitHub Releases

The workflow refuses to publish if that tag or GitHub Release already exists.


## License & credits

Released under the [GNU General Public License v3.0](LICENSE).

Written by Nick Jennings.

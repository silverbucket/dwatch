# dwatch

Track disk space growth over time. Takes periodic snapshots of directory sizes and surfaces what is growing — by absolute amount, percentage, or daily rate.

Works on macOS (primary) and Linux.

---

## How it works

`dwatch scan` walks the filesystem from a root directory up to a configurable depth, computes the total allocated size of each directory (using block counts, not logical file size — sparse VM disk images report correctly), and saves a timestamped JSON snapshot to `~/.dwatch/`.

The reporting commands (`diff`, `top`, `status`, `alert`) compare snapshots to show what changed. Snapshots are plain JSON and can be deleted or moved freely.

---

## Install

```sh
git clone <repo>
cd dwatch
make install        # builds and copies binary to /usr/local/bin/dwatch
```

Requires Go 1.21 or later.

To uninstall:

```sh
make uninstall
```

---

## Quick start

```sh
# Take a snapshot
dwatch scan

# See what changed since the previous scan
dwatch status

# See what grew in the last week
dwatch diff --since 1w

# Rank directories by growth this month
dwatch top --since 1m

# Rank by percentage growth — catches small directories ballooning in size
dwatch top --since 1d --by pct
```

---

## Cron setup

The intended workflow is to run `scan` automatically, then query the snapshots on demand or via a separate alert job.

Open your crontab with `crontab -e` and add:

```
# Snapshot every hour, silently
0 * * * *   /usr/local/bin/dwatch scan --quiet

# Alert at 8am if any directory grew more than 500 MB overnight
# cron emails output to MAILTO when the command exits non-zero
0 8 * * *   /usr/local/bin/dwatch alert --since 1d --threshold 500mb

# Weekly report every Monday at 9am
0 9 * * 1   /usr/local/bin/dwatch top --since 1w --limit 20
```

Set `MAILTO` in your crontab to receive alerts by email:

```
MAILTO=you@example.com
```

---

## Commands

### `dwatch scan`

Walk the filesystem and save a snapshot.

```sh
dwatch scan [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-r, --root` | `/` | Root directory to scan from |
| `-n, --depth` | `5` | Maximum directory depth to track |
| `-s, --skip` | see below | Paths to exclude (repeatable) |
| `--show` | `10` | Number of largest directories to print after scan (`0` = all) |
| `-q, --quiet` | false | Suppress output; write a one-line summary to stderr instead (use in cron) |

**Default skip paths (macOS):** `/dev`, `/System/Volumes`, `/net`, `/home`, `~/.orbstack`

**Default skip paths (Linux):** `/proc`, `/sys`, `/dev`, `/run`

Examples:

```sh
dwatch scan --root /Users --depth 4
dwatch scan --root / --skip /Volumes/BackupDrive
dwatch scan --quiet   # for cron use
dwatch scan --show 25
```

---

### `dwatch status`

Print a quick overview: the latest snapshot's largest directories and what grew since the previous scan.

```sh
dwatch status
```

No flags beyond `--data-dir`.

---

### `dwatch diff`

Show a change table comparing two snapshots.

```sh
dwatch diff [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-s, --since` | — | Compare against the nearest snapshot before this time. If omitted, compares against the previous snapshot. |
| `--min-change` | `1mb` | Minimum absolute change to include in output |
| `-l, --limit` | `30` | Maximum rows to show (`0` = all) |
| `-a, --all` | false | Show all directories including unchanged ones |

The `--since` flag accepts:

| Format | Example | Meaning |
|--------|---------|---------|
| Hours | `6h` | 6 hours ago |
| Days | `3d` | 3 days ago |
| Weeks | `2w` | 2 weeks ago |
| Months | `1m` | 1 month ago |
| Date | `2026-05-01` | That specific date |

Examples:

```sh
dwatch diff                      # latest vs previous snapshot
dwatch diff --since 1w
dwatch diff --since 2026-05-01 --min-change 100mb
dwatch diff --all --limit 0      # show every tracked directory
```

---

### `dwatch top`

Rank directories by growth within a time window.

```sh
dwatch top [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-s, --since` | — | Time window (same formats as `diff`). If omitted, compares against the previous snapshot. |
| `--by` | `growth` | Sort order: `growth` (absolute bytes), `pct` (percentage), `rate` (bytes/day) |
| `-l, --limit` | `20` | Number of results (`0` = all) |

**Sort modes:**

- `growth` — ranks by total bytes added. Best for finding the biggest absolute consumers.
- `pct` — ranks by percentage growth. Best for catching small directories that are growing rapidly (e.g. a cache that went from 200 MB to 1.5 GB is ranked above a 500 GB directory that grew 5%).
- `rate` — ranks by bytes per day, normalized across the time window.

Examples:

```sh
dwatch top --since 1w
dwatch top --since 1d --by pct    # surface fast-growing small dirs
dwatch top --since 1m --by rate --limit 10
```

---

### `dwatch alert`

Exit with code 1 if any directory grew beyond a threshold. Designed for cron and scripting.

```sh
dwatch alert --threshold <size> [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-t, --threshold` | `500mb` | Growth threshold; required. Accepts `kb`, `mb`, `gb`, `tb`. |
| `-s, --since` | — | Comparison window. If omitted, compares against the previous snapshot. |

When any directory exceeds the threshold, `alert` prints a table of offenders and exits 1. If nothing exceeds the threshold it exits 0 and prints nothing — safe to use in scripts.

Examples:

```sh
dwatch alert --threshold 1gb --since 1d
dwatch alert --threshold 500mb          # compare to previous scan
```

---

## Global flags

Available on all commands:

| Flag | Default | Description |
|------|---------|-------------|
| `-d, --data-dir` | `~/.dwatch` | Directory where snapshots are stored |

---

## Snapshot storage

Snapshots are saved as JSON files in `~/.dwatch/` (or `--data-dir`):

```
~/.dwatch/
  snap_20260512_090000.json
  snap_20260513_090000.json
  ...
```

Each file is self-contained and human-readable. To prune old snapshots, delete files directly. There is no database to maintain.

---

## macOS notes

**Sparse files and VM disk images:** `dwatch` uses block-count sizing (equivalent to `du`) rather than logical file size. This means OrbStack VM disks, Docker overlayfs images, and other sparse files report their actual on-disk usage rather than their virtual size.

**OrbStack:** `~/.orbstack` is in the default macOS skip list. OrbStack's virtual filesystem can contain millions of files and scanning it takes hours. Skip it.

**Hard links:** Multiply-linked files (common in the macOS system directory) are counted once per unique inode, consistent with `du` behaviour.

**`/System/Volumes`:** Skipped by default to prevent double-counting APFS firmlinks. User data under `/Users` is accessible via the firmlink at the root level.

**Mounted volumes:** `/Volumes` is not skipped by default. If you have slow or large external volumes mounted there, add them to `--skip`.

---

## Linux notes

Default skip list: `/proc`, `/sys`, `/dev`, `/run`. Add bind-mounted paths or network filesystems as needed via `--skip`.

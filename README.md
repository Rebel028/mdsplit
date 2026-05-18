# mdsplit

A CLI tool that splits a monolithic markdown document into multiple files based on ATX headings (`#`, `##`, etc.).

## Installation

### Download a binary

Grab the latest pre-built binary from the [Releases](https://github.com/Rebel028/mdsplit/releases) page:

| Platform | File |
|----------|------|
| Linux (x86-64) | `mdsplit-linux-amd64` |
| Windows (x86-64) | `mdsplit-windows-amd64.exe` |
| macOS (Apple Silicon) | `mdsplit-darwin-arm64` |

### Install with Go

```bash
go install github.com/Rebel028/mdsplit/cmd/mdsplit@latest
```

### Build from source

```bash
git clone https://github.com/Rebel028/mdsplit.git
cd mdsplit
go build -o mdsplit ./cmd/mdsplit
```

## Usage

```
mdsplit [flags] <input-file>
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--max-level` | `-l` | `1` | Maximum heading level to split on (1–6) |
| `--table-of-contents` | `-t` | `false` | Generate a `toc.md` file |
| `--navigation` | `-n` | `false` | Append a navigation footer to each page |
| `--output` | `-o` | `.` | Output directory (created if it does not exist) |
| `--force` | `-f` | `false` | Allow writing into an existing output directory |
| `--verbose` | `-v` | `false` | Print detailed processing logs |
| `--numbered` | `-u` | `false` | Prefix filenames with section numbers; use ordered lists in the TOC |

### Examples

Split by top-level headings, writing files to the current directory:

```bash
mdsplit document.md
```

Split by headings up to level 3, generate a TOC, add navigation links, and write to `./output/`:

```bash
mdsplit -l 3 -t -n -o output/ document.md
```

Split with numbered file prefixes (e.g. `1-introduction.md`, `1-1-background.md`):

```bash
mdsplit -l 2 -u -o output/ document.md
```

### Output

Each heading at or above `--max-level` becomes a separate `.md` file named after the heading text (lowercased, spaces replaced with hyphens, special characters removed). Content before the first matching heading is written to `introduction.md`.

When `--table-of-contents` is enabled, a `toc.md` (or `0-toc.md` with `--numbered`) is generated with links to all split files.

## Project layout

```
.
├── cmd/
│   └── mdsplit/       # CLI entry point
├── internal/
│   └── splitter/      # Core splitting logic
├── go.mod
└── go.sum
```

## License

MIT

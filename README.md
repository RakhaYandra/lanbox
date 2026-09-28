# LANBox

Local-first file sharing over the LAN. HTTPS always on (per-boot
self-signed cert — verify the console fingerprint on first connect),
token + PIN auth, no cloud, no account, no internet.

## Install

```sh
make build        # -> bin/lanbox (GOOS=linux|darwin|windows)
```

## Usage

```sh
# Terminal 1: serve a directory
bin/lanbox serve --dir ~/LANBox --port 8080

# Terminal 2: serve the web UI build (from lanbox-web repo)
bin/lanbox serve --dir ~/LANBox --web-dir ../lanbox-web/dist

# Browser: http://<lan-ip>:8080  |  API: /api/v1/info, /files, /files/download, /files/upload
bin/lanbox version
```

## Checks

```sh
make fmt     # must print nothing
make vet
make test
```

Docs (requirements, architecture, guides + PDFs): `lanbox-docs` repo.
Web UI source: `lanbox-web` repo.

## Benchmark (loopback, legion5, 2026-09-27)

| Case | Result |
|---|---|
| 1 GB upload | 2.4s (~440 MB/s), server RSS ~140 MB |
| CLI send 100 MB | ~434 MB/s + `Verified` |
| 2 / 4 parallel uploads | 2×201 / 4×201 |
| 8 parallel uploads | 4×201 + 4×429 (semaphore working) |
| 8×20 MB parallel RSS | ~390 MB (multipart buffers, bounded) |
| `--limit 5MB/s` | measured ~5.5 MB/s (±10%) |

Memory stays flat versus file size; concurrency bounded by the 4-slot
semaphore. Rerun: `./test/security.sh` and `./test/load.sh` (see IMPLEMENTATION §6).

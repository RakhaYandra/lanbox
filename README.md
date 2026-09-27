# LANBox

Local-first file sharing over the LAN. No cloud, no account, no internet.

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

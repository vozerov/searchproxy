# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- A side closing its connection is now passed on to the other side as FIN
  (and a reset or write error closes both). Previously the proxy waited for
  both directions to finish, so a backend dropping an idle connection stayed
  invisible to the client, and keepalive pools (nginx `keepalive`) reused the
  dead connection until their read timeout.

## [0.1.0] - 2026-05-01

### Added
- TCP proxy that forwards connections to upstream targets resolved through
  the system DNS resolver (libc / `getaddrinfo`), honoring the `search`
  list from `/etc/resolv.conf`.
- Optional static HTTP file server.
- `/healthz` (liveness) and `/readyz` (ready when every TCP listener is
  accepting) endpoints, default port `:8081`.
- YAML config with `proxies`, `static`, `health` sections; configurable via
  `-config` flag (default `/etc/searchproxy/config.yaml`).
- Multi-arch container image (`linux/amd64`, `linux/arm64`) built on Alpine
  with `CGO_ENABLED=1` + `GODEBUG=netdns=cgo`.
- GitHub Actions workflow that builds and pushes images to
  `quay.io/vozerov/searchproxy` on `main` and `v*` tags.
- Examples: standalone configs and a Kubernetes manifest set.

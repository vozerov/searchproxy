# searchproxy

A tiny TCP proxy in Go that forwards connections to upstream targets resolved
through the system DNS resolver. Built for one specific job: forwarding TCP
traffic to Kubernetes services using **short DNS names** (`redis`,
`postgres`, `kafka-broker`) that get expanded against the pod's
`/etc/resolv.conf` `search` list.

It can also serve a static directory over HTTP — handy for shipping in-pod
docs or a tiny dashboard alongside the proxy.

## Why this exists

Kubernetes injects a `search` list into every pod's `/etc/resolv.conf`:

```
search my-namespace.svc.cluster.local svc.cluster.local cluster.local
options ndots:5
```

That's what makes `curl http://redis:6379` work from inside a pod — libc's
`getaddrinfo` walks the search list and resolves `redis` to
`redis.my-namespace.svc.cluster.local`.

### Why not HAProxy or Nginx?

Both bypass libc's resolver and bring their own — and neither honors `search`
the way `getaddrinfo` does:

- **Nginx** requires the `resolver` directive, expects an FQDN in
  `proxy_pass`, and its built-in resolver ignores `search` and `ndots` in
  `/etc/resolv.conf`. You end up hard-coding the namespace in every upstream.
- **HAProxy** has its own `resolvers` section. It will read nameservers from
  `/etc/resolv.conf`, but the `search` list is not appended to short names —
  same problem, you write `service.namespace.svc.cluster.local` everywhere.

If you ever moved the same workload between namespaces, you know how brittle
that gets. With `searchproxy` you write `target: redis:6379` once and the
config travels with the pod — the search path makes the resolution
namespace-aware for free.

## What it does

- Listens on N TCP ports defined in YAML.
- For each accepted connection, dials the configured target (a short or
  fully-qualified DNS name) using Go's `net.Dial`, which goes through the
  system resolver and honors the `search` list.
- Optionally serves a static directory over HTTP.
- Exposes `/healthz` and `/readyz` for Kubernetes probes.

That's it. Around 100 lines of Go.

## Configuration

```yaml
# /etc/searchproxy/config.yaml

proxies:
  - listen: ":5432"
    target: "postgres:5432"   # resolved against the pod's search list
  - listen: ":6379"
    target: "redis:6379"
  - listen: ":9092"
    target: "kafka-broker.kafka:9092"  # FQDNs work too

health:
  listen: ":8081"             # default; set to "" to disable

static:                       # optional
  listen: ":8080"
  dir: "/var/www/docs"
```

Run:

```bash
searchproxy -config /etc/searchproxy/config.yaml
```

## Health endpoints

| Endpoint   | Meaning                                                           |
| ---------- | ----------------------------------------------------------------- |
| `/healthz` | 200 once the process is running. Use as `livenessProbe`.          |
| `/readyz`  | 200 only after every configured TCP listener is accepting. Use as `readinessProbe`. |

Default port is `:8081`. Override with `health.listen`.

## Building

```bash
go build -o searchproxy .
```

For full libc / NSS parity (recommended in production — matches
`getaddrinfo` exactly, including `/etc/nsswitch.conf` and NSS modules):

```bash
CGO_ENABLED=1 GODEBUG=netdns=cgo go build -o searchproxy .
```

The pure Go resolver also reads `/etc/resolv.conf` and honors the `search`
list, so for plain Kubernetes service discovery the difference rarely
matters. The shipped container image uses cgo + glibc.

## Container image

```
quay.io/vozerov/searchproxy:latest
```

Built from `Dockerfile` against `alpine:3.20` (musl) with
`GODEBUG=netdns=cgo` baked in — resolution goes through musl's
`getaddrinfo`, which honors `search` and `ndots` from `/etc/resolv.conf`.
Multi-arch (amd64 / arm64).

## Examples

See [`examples/`](./examples) for:

- A minimal config (`examples/basic.yaml`)
- A config with the static file server (`examples/with-static.yaml`)
- A full Kubernetes manifest set (`examples/kubernetes/`)

## License

MIT

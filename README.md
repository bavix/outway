# Outway

Outway is a small service that steers egress traffic by domain. Policy is decided at the application layer (L7) using domain names from DNS, while enforcement is done at the network layer (L3/L4) by marking IPs and letting the OS route them (multi‑WAN by domains).

Example: you have two uplinks (wan1, wan2). You want `foo.example.com` via `wan1`, while `bar.example.com` always goes via `wan2`. With Outway you configure domain rules and the system enforces them.

## How it works

- Outway handles DNS queries for your apps
- On each DNS answer, it extracts IPs, assigns a mark with TTL, and programs the OS firewall
- Marked IPs follow the route/interface mapped to the matching domain rule group
- When TTL expires, the mark is removed automatically

## Features

- Domain‑based egress routing (interface per rule group)
- Request coalescing: deduplicates concurrent cache misses for the same host/QTYPE
- Clean URL format for upstream resolvers (single, strict format):
  - `udp://host[:port]` (default 53)
  - `tcp://host[:port]` (default 53)
  - `tls://host[:port]` or `dot://host[:port]` (default 853)
  - `quic://host[:port]` or `doq://host[:port]` (default 853)
  - `https://host/path` (DoH, RFC 8484)
- Built‑in Admin UI with WebSocket realtime updates and polling fallback
- Prometheus metrics at `/metrics`
- Health endpoint at `/health`

## Caching

- LRU cache keyed by `fqdn:qtype` with per‑record TTL respected
- Expired entries are evicted on read; fresh responses are cached
- Singleflight coalescing prevents upstream stampedes for identical in‑flight queries

## Quick start

1) Install

```bash
go install github.com/bavix/outway@latest
```

2) Configure

Use `config.test.yaml` as a reference. Minimal realistic example (aligned with the test config):

```yaml
app_name: outway

listen:
  udp: ":53"
  tcp: ":53"

upstreams:
  - name: cloudflare-doh
    address: https://cloudflare-dns.com/dns-query
    weight: 1
  - name: cf-ipv4
    address: udp://1.1.1.1:53
    weight: 1

rule_groups:
  - name: Default
    description: Default egress group
    via: utun4            # interface name
    pin_ttl: true         # keep TTL from DNS, don't shrink aggressively
    patterns:
      - "*.example.com"
```

Multi‑WAN example:

```yaml
rule_groups:
  - name: wan1-sites
    via: wan1
    pin_ttl: true
    patterns:
      - "foo.example.com"

  - name: wan2-sites
    via: wan2
    pin_ttl: true
    patterns:
      - "bar.example.com"
```

Fuller example (same structure as `config.test.yaml`):

```yaml
app_name: outway

listen:
  udp: ":53"
  tcp: ":53"

upstreams:
  - name: cloudflare-doh
    address: https://cloudflare-dns.com/dns-query
    weight: 1
  - name: cf-ipv4
    address: udp://1.1.1.1:53
    weight: 1
  - name: cf-ipv6
    address: udp://[2606:4700:4700::1111]:53
    weight: 1
  - name: google
    address: udp://8.8.8.8:53
    weight: 1
  - name: opendns
    address: udp://208.67.222.222:53
    weight: 1

rule_groups:
  - name: "YouTube & Google Services"
    description: Route YouTube and Google services through specific interface
    via: utun4
    patterns:
      - "*.youtube.com"
      - "*.googlevideo.com"
      - "*.ytimg.com"
      - "*.googleapis.com"
      - "*.googleusercontent.com"
    pin_ttl: true

  - name: Social Media
    description: Social media platforms routing
    via: utun4
    patterns:
      - "*.instagram.com"
      - "*.facebook.com"
      - "*.twitter.com"
      - "*.x.com"
      - "*.tiktok.com"
      - "*.snapchat.com"
    pin_ttl: true

  - name: Streaming Services
    description: Video streaming platforms
    via: utun4
    patterns:
      - "*.netflix.com"
      - "*.hulu.com"
      - "*.disney.com"
      - "*.amazon.com"
      - "*.twitch.tv"
    pin_ttl: true

  - name: Development Tools
    description: Development and version control services
    via: utun3
    patterns:
      - "*.github.com"
      - "*.gitlab.com"
      - "*.bitbucket.org"
      - "*.docker.com"
      - "*.npmjs.org"
    pin_ttl: true

  - name: Blocked Domains
    description: Blocked malicious domains
    via: lo0
    patterns:
      - "*.malware.com"
      - "*.phishing-site.com"
      - "*.ads-tracker.com"
    pin_ttl: true

history:
  enabled: true
  max_entries: 10000

log:
  level: info

cache:
  enabled: true
  max_entries: 20000

http:
  enabled: true
  listen: 127.0.0.1:47823
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 2m0s
  max_header_bytes: 1048576

hosts:
  - pattern: localhost
    a:
      - 127.0.0.1
    ttl: 60
  - pattern: "*.example.com"
    a:
      - 127.0.0.1
    ttl: 60
```

3) Run

```bash
outway run --config ./config.yaml
```

- Admin UI: `http://127.0.0.1:47823/`
- Metrics: `http://127.0.0.1:47823/metrics`
- Health: `http://127.0.0.1:47823/health`

## Commands

- `outway run` - Start the DNS proxy service
- `outway cleanup` - Cleanup all firewall rules created by Outway
- `outway self-update` - Update to the latest version from GitHub
- `outway --version` - Show version information

### Self-update

Update Outway to the latest version:

```bash
# Update to latest stable version
outway self-update

# Include prerelease versions
outway self-update --prerelease
```

The self-update command will download the appropriate binary for your platform, replace the current binary, and exit with code 42 to trigger automatic restart by your init system (systemd, OpenWrt /etc/init.d, etc.).

## Admin UI & configuration

- UI: `http://127.0.0.1:47823/` (by default)
- WebSocket realtime is used automatically; if WS is unavailable, UI switches to polling

To run the admin UI on a different address/port, configure the `http` section in the config:

```yaml
http:
  enabled: true
  listen: 0.0.0.0:8080   # address:port for Admin UI and API
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 2m0s
```

Upstreams in YAML are specified only in URL format — the type is derived from the scheme (`udp://`, `tcp://`, `dot://`, `doq://`, `https://`).

## Observability

- `/metrics` exposes Prometheus metrics (query rate, latency, marks, etc.)
- UI Dashboard shows:
  - Uptime
  - Queries in the last minute and error count (realtime)
  - Cache hit rate (when available)

## System backends

- Linux: nftables/iptables
- macOS: pf

## Build

```bash
make build
make lint
```

# tv-guide

Spokane HDHomeRun XMLTV guide provider in Go. Discovers channels from your HDHomeRun, pulls SiliconDust XMLTV guide data, enriches with a Spokane station overlay, and serves Jellyfin-compatible XMLTV. Video still comes from Jellyfin’s HDHomeRun tuner — this service is guide-only.

On startup it loads a cached guide if present, then refreshes in the background with random jitter between `REFRESH_MIN_HOURS` and `REFRESH_MAX_HOURS` (SiliconDust requires non-fixed scheduling). Each refresh reads fresh `DeviceAuth` from the tuner, downloads gzip XMLTV from SiliconDust, optionally filters to favorites, applies Spokane overlays, and writes the cache atomically. Failed refreshes keep serving stale data; `/status` reports last success and errors.

## HTTP endpoints

| Path | Description |
|------|-------------|
| `GET /xmltv.xml` | XMLTV document for Jellyfin |
| `GET /health` | Liveness (`ok`) |
| `GET /status` | JSON refresh/cache status |
| `GET /channels` | JSON lineup with `xmltvChannelId` |

## Environment

| Variable | Default |
|----------|---------|
| `HDHR_BASE_URL` | `http://10.0.30.99` |
| `PORT` | `8080` |
| `REFRESH_MIN_HOURS` | `20` |
| `REFRESH_MAX_HOURS` | `28` |
| `FAVORITES_ONLY` | `false` |
| `CACHE_PATH` | `cache/xmltv.xml` (Compose uses `/data/cache/xmltv.xml`) |
| `HDHR_ACCOUNT_EMAIL` | _(empty)_ — optional SiliconDust account email; uses Email+DeviceIDs with the tuner’s DeviceID |

### SiliconDust HTTP 403

`api.hdhomerun.com` rate-limits frequent XMLTV downloads. This service:

- Refreshes only from a **background loop** (not on each Jellyfin `/xmltv.xml` poll)
- Coalesces concurrent refreshes (single-flight)
- After a 403, retries in **15–30 minutes** instead of waiting a full day

If you still see 403 after deploy: wait a few minutes, ensure a single `tv-guide` replica, and confirm the HDHomeRun can reach SiliconDust (guide sync). Optional: set `HDHR_ACCOUNT_EMAIL` to your my.hdhomerun.com account email.

## Local development (asdf)

Go is pinned in [`.tool-versions`](.tool-versions) to **1.27.3**:

```bash
asdf install          # installs golang 1.27.3
asdf reshim golang
go test ./...
go build -o bin/tv-guide ./cmd/tv-guide
./bin/tv-guide
```

## Docker / Komodo

```bash
docker compose up --build
```

[`compose.yaml`](compose.yaml) exposes port `8080` and persists the guide cache under `/data/cache`. For Komodo, create a Stack pointing at this repo’s `compose.yaml` (or host copy), set `HDHR_BASE_URL` if needed, and deploy on a server that can reach the HDHomeRun.

**Network:** the container must reach `10.0.30.99`. If Docker bridge cannot, use `network_mode: host`, macvlan, or a route into that subnet.

## Jellyfin setup

1. Keep the existing **HDHomeRun** tuner (`http://10.0.30.99`).
2. Dashboard → Live TV → Guide Data Providers → add **XMLTV**.
3. XMLTV URL:
   - Same Compose network: `http://tv-guide:8080/xmltv.xml`
   - Host port: `http://<komodo-host>:8080/xmltv.xml`
4. Refresh Guide Data → **Map Channels** (match by GuideNumber / display name).
5. Do not enable Schedules Direct at the same time — Jellyfin allows only one guide provider type.

## Fixtures

`fixtures/discover.json` and `fixtures/lineup.json` were captured from the live HDHomeRun at `10.0.30.99` (`DeviceAuth` redacted). `fixtures/xmltv-sample.xml` is a synthetic SiliconDust-shaped snippet for tests. See [`fixtures/README.md`](fixtures/README.md).

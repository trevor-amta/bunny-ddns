# Bunny Dynamic DNS Updater (v0.1.3)

A tiny Go service that keeps one or more Bunny.net DNS records in sync with the latest public IP address.

## Features (v0.1.3)
- Polls multiple WAN IP endpoints on a configurable interval
- Verifies the live Bunny DNS record value before updating, so restarts avoid redundant writes
- Fails fast if a configured record ID is missing from the Bunny zone
- Updates specific Bunny DNS records only when the IP changes
- Stateless design: restart the container without losing state
- Ships as a minimal Docker image suitable for Coolify or other platforms

## Getting Started
1. Copy `.env.example` to `.env` and fill in your Bunny credentials and record IDs.
2. Build and run locally with Docker Compose:
   ```bash
   docker compose up --build
   ```
3. Inspect logs to confirm IP detection and DNS update behavior.

## Configuration
All settings are supplied via environment variables:

- `BUNNY_API_KEY` – Bunny API access key with permission to edit DNS records
- `BUNNY_ZONE_ID` – Target DNS zone identifier
- `BUNNY_RECORDS_JSON` – JSON array describing the records to update (e.g., `[{"id":123,"name":"home","type":"A"}]`). The loader now tolerates escaped JSON strings such as `[{\"id\":123}]`, which some platforms (like Coolify) inject. Use an empty string (`""`) for the `name` field to target the zone apex (root record); the Bunny client falls back to listing records when the per-record endpoint is unavailable (HTTP 405).
- `POLL_INTERVAL_SECONDS` – Seconds between WAN IP checks (default 120)
- `WAN_IP_ENDPOINTS` – Comma-separated list of HTTPS endpoints used to detect the WAN IP (defaults to api.ipify.org, ipv4.icanhazip.com)
- `USER_AGENT` – Custom user agent string for outbound HTTP requests

## Deployment
- **Coolify**: Point a service at this repo, set environment variables in the dashboard, and deploy the container.
- **Other platforms**: Use the provided Dockerfile to build a tiny image and run it as a background worker or scheduled job anywhere Go binaries are supported.

## Roadmap
- Add configurable retry/backoff logic for Bunny API failures
- Provide pre-built multi-arch images via CI
- Optional status/metrics endpoint for health checks
- Extend configuration to allow record discovery by name instead of numeric ID

## License
MIT © 2025 AMTA Management LLC

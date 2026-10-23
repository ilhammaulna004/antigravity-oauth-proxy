# Antigravity OAuth Proxy

Antigravity OAuth Proxy makes the models available to your Google Antigravity account usable through the Gemini API, an OpenAI-compatible endpoint, or MCP. It handles Google OAuth credentials and translates requests to the internal Cloud Code API used by Antigravity.

The MCP server at `/mcp` gives agents tools to discover models available to your Antigravity account and send one-shot prompts without configuring a Gemini or OpenAI API client.

```text
  ┌───────────────┐          ┌───────────────────┐          ┌───────────────────────┐
  │ External Tool │          │ Antigravity Proxy │          │ Google Cloud Endpoint │
  │ (OpenCode/etc)│          │ (Local or Worker) │          │      (Cloud Code)     │
  └───────┬───────┘          └─────────┬─────────┘          └───────────┬───────────┘
          │                            │                                │
          │  API or MCP request        │    Cloud Code request          │
          │ ─────────────────────────▶ │ ─────────────────────────────▶ │
          │                            │    OAuth access token          │
          │                            │                                │
          │  API or MCP response       │    Cloud Code response         │
          │ ◀───────────────────────── │ ◀───────────────────────────── │
          │    JSON or SSE stream      │                                │
          │                            │                                │
          ▼                            ▼                                ▼
```

Use the native Gemini endpoint when your client supports it. The OpenAI-compatible endpoint is available for clients that only speak the OpenAI protocol.

## Web Dashboard

The proxy includes a built-in web dashboard at `/dashboard/` for monitoring and management. The Vue 3 frontend is compiled and embedded directly into the Go binary — no separate Node.js runtime is needed at deploy time.

**Dashboard features:**

- **Overview** — Real-time request stats, usage charts, and top models at a glance
- **Playground** — Send prompts to any available model and view streamed responses interactively
- **Requests** — Browse and filter all proxied API requests with full detail modals
- **Models** — Search and explore all models available to the signed-in account
- **Accounts** — View connected Google OAuth accounts and their token status
- **Security** — Change the dashboard password and review active session details

| Overview | Playground | Models |
| :---: | :---: | :---: |
| ![Overview](docs/screenshots/dashboard_overview.jpg) | ![Playground](docs/screenshots/dashboard_playground.jpg) | ![Models](docs/screenshots/dashboard_models.jpg) |

The dashboard is protected by password authentication. Default credentials are `admin` / `admin` — change the password on first login via the Security page.

## Quick start

### From source

```bash
git clone https://github.com/dvcrn/antigravity-oauth-proxy.git
cd antigravity-oauth-proxy

# Build and run
make build
ADMIN_API_KEY="replace-with-a-long-random-value" ./antigravity-oauth-proxy
```

### Using Docker Compose

Install Docker and Docker Compose on the host PC. Create a `.env` file in the repository directory with your own long, random key (do not use the Compose default):

```dotenv
ADMIN_API_KEY=replace-with-a-long-random-value
```

```bash
docker compose up -d --build
```

Open `http://localhost:9878/dashboard/` on the host PC. Sign in with `admin` / `admin` and change the password on the **Security** page before making the proxy public. Account data and usage history persist in `~/.config/antigravity-oauth-proxy` on the host. To check logs, run `docker compose logs -f antigravity-proxy`.

### Access from outside via Cloudflare Tunnel (home PC)

This runs the app on your PC and exposes it through Cloudflare Tunnel; it does **not** deploy the app to Cloudflare Workers. You need a domain managed by Cloudflare and `cloudflared` running on the same host as Docker. No router port forwarding is needed.

1. Start the proxy with Docker Compose as above. Before publishing it, set a unique `ADMIN_API_KEY` in `.env` and change the dashboard's default `admin` password. For host-only access to the proxy port, change the `9878` port mapping in `docker-compose.yml` from `"9878:9878"` to `"127.0.0.1:9878:9878"`, then run `docker compose up -d --build` again.
2. In Cloudflare Zero Trust, go to **Networks → Tunnels**, create a Cloudflare Tunnel, and follow the displayed instructions to install and run `cloudflared` on the host PC. Keep the connector running for as long as you want remote access.
3. Add a **Public Hostname** such as `proxy.example.com` to the tunnel. Set service type to **HTTP** and URL to `localhost:9878` (or `http://localhost:9878`, depending on the form). Cloudflare provides HTTPS for the public hostname. Test `https://proxy.example.com/dashboard/` from outside your home network.

Use `https://proxy.example.com/v1beta` for Gemini clients or `https://proxy.example.com/v1` for OpenAI-compatible clients; send your `ADMIN_API_KEY` as a bearer token or `X-Goog-Api-Key`. The `/v1/models` endpoint is public, and the dashboard has its own password: **a tunnel does not add authentication**. Consider Cloudflare Access for the dashboard; if you protect the same hostname with Access, API clients must also satisfy its Access policy. Never publish the default dashboard password or Compose API key.

Google OAuth redirects to `http://localhost:51121/oauth-callback`, not your public hostname. With Docker or a remote browser, the automatic callback may not reach the app. Copy the final localhost callback URL (or authorization code) from your browser address bar and paste it into **Connecting remotely or callback blocked?** in the dashboard's **Connect Account** dialog. Port `51121` does not need to be routed through the tunnel.

### Using Go install

```bash
go install github.com/dvcrn/antigravity-oauth-proxy/cmd/antigravity-oauth-proxy@latest
```

### Using mise

```bash
mise use -g go:github.com/dvcrn/antigravity-oauth-proxy/cmd/antigravity-oauth-proxy@latest
```

The server listens on `http://localhost:9878` by default.

### Connecting Google Accounts

Open the web dashboard at `http://localhost:9878/dashboard/accounts` (default admin credentials: `admin` / `admin`). Click **"Connect Account"** to sign in with your Google Antigravity account directly in the browser. You can connect multiple Google accounts for automatic quota failover.

Test the native Gemini endpoint with:

```bash
curl "http://localhost:9878/v1beta/models/gemini-3-flash:generateContent" \
  -H "X-Goog-Api-Key: replace-with-your-admin-key" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [{"role": "user", "parts": [{"text": "Say hello in one sentence."}]}]
  }'
```

## Client setup

For Gemini clients, use `http://localhost:9878/v1beta` as the base URL. For OpenAI-compatible clients, use `http://localhost:9878/v1`.

An OpenCode provider using the native Gemini protocol looks like this:

```json
{
  "provider": {
    "antigravity": {
      "npm": "@ai-sdk/google",
      "name": "Antigravity",
      "options": {
        "baseURL": "http://localhost:9878/v1beta",
        "apiKey": "replace-with-your-admin-key"
      },
      "models": {
        "gemini-3-flash": {
          "name": "Gemini 3 Flash (Antigravity)"
        }
      }
    }
  }
}
```

Query `GET /v1/models` to see the models available to the signed-in account.

## Authentication

Google accounts are managed seamlessly via the Web Dashboard at `/dashboard/accounts`. The proxy automatically stores accounts in `~/.config/antigravity-oauth-proxy/accounts/`, refreshes expired tokens, and automatically discovers each account's GCP Project ID.

Set `CLOUDCODE_OAUTH_CREDS_PATH` to use a custom credentials file, or provide the complete credentials JSON through `CLOUDCODE_OAUTH_CREDS` in environments without persistent disk storage. You can set `CLOUDCODE_GCP_PROJECT_ID` when automatic project discovery is unsuitable.

`ADMIN_API_KEY` protects generation, admin, and MCP requests. Clients may send it as a bearer token or an `X-Goog-Api-Key` header. The `/v1/models` endpoint is public.

## Endpoints

### Proxy API

| Endpoint | Purpose |
| --- | --- |
| `POST /v1beta/models/{model}:generateContent` | Gemini-compatible non-streaming generation |
| `POST /v1beta/models/{model}:streamGenerateContent` | Gemini-compatible streaming generation |
| `POST /v1/chat/completions` | OpenAI-compatible chat completions |
| `GET /v1/models` | Models available to the signed-in account |
| `POST /mcp` | Stateless MCP server with `ask_gemini` and `ask_gemini_models` |

### Dashboard

| Endpoint | Purpose |
| --- | --- |
| `GET /dashboard/` | Web dashboard UI |
| `POST /api/auth/login` | Dashboard authentication |
| `POST /api/auth/change-password` | Change dashboard password |
| `GET /api/accounts` | List connected OAuth accounts |
| `POST /api/accounts/test` | Test an account's connectivity |
| `GET /api/usage/stats` | Aggregated usage statistics |
| `GET /api/usage/requests` | Paginated request history |
| `POST /api/playground/chat` | Playground chat completions |

### Workers Admin

| Endpoint | Purpose |
| --- | --- |
| `POST /admin/auth/start` | Start Workers Google authorization |
| `GET /admin/auth/status` | Read the Workers authorization state |
| `POST /admin/auth/status` | Exchange an authorization code and store tokens |
| `POST /admin/tokens` | Store OAuth tokens manually in Workers KV |
| `GET /admin/status` | Check whether Workers credentials are configured |

## MCP clients

The `/mcp` endpoint uses stateless streamable HTTP with JSON responses. It keeps no conversation or session state between calls.

MCP requests use the proxy's Google OAuth credentials upstream and the same `ADMIN_API_KEY` as the generation endpoints. No separate Gemini API key is required.

MCP configuration varies by client. Configure a streamable HTTP server with:

| Setting | Value |
| --- | --- |
| URL | `http://localhost:9878/mcp` |
| Header | `Authorization: Bearer replace-with-your-admin-key` |

For clients that use an `mcpServers` JSON object:

```json
{
  "mcpServers": {
    "ask-antigravity": {
      "type": "http",
      "url": "http://localhost:9878/mcp",
      "headers": {
        "Authorization": "Bearer replace-with-your-admin-key"
      }
    }
  }
}
```

The client discovers these tools after it connects:

| Tool | Input | Result |
| --- | --- | --- |
| `ask_gemini_models` | None | The default model and available model IDs |
| `ask_gemini` | `model`, `prompt` | The requested model, model that served the request, and response text |

Call `ask_gemini_models` first when the model ID is not already known. Its results reflect the models currently available to the signed-in Antigravity account.

`ask_gemini` is one-shot. It does not retain conversation history, so `prompt` must include all context needed for that call. The returned `model` may differ from `requested_model` when the proxy resolves a model variant or falls back after an upstream 404.

## macOS LaunchAgent

Run the proxy as a background service that starts automatically on login:

```bash
./install-launchagent.sh
```

The script creates a LaunchAgent plist, symlinks it to `~/Library/LaunchAgents/`, and starts the service immediately. It auto-restarts after crashes and logs output to `~/Library/Logs/antigravity-oauth-proxy.log`.

To uninstall:

```bash
./uninstall-launchagent.sh
```

## Cloudflare Workers

Workers deployments store OAuth credentials in KV and use a [Workers VPC](https://developers.cloudflare.com/workers-vpc/) tunnel for provider egress. Install Go 1.25 or newer, [mise](https://mise.jdx.dev/), and Wrangler 4, then run `mise install` and `wrangler login`. The account, namespace, and tunnel IDs checked into `wrangler.toml` belong to the maintainer deployment and must be replaced for another Cloudflare account.

1. Create a tunnel in **Cloudflare Dashboard > Workers VPC > Tunnels** and run `cloudflared` on a machine with normal Internet access.
2. Create a KV namespace with `wrangler kv namespace create ANTIGRAVITY_OAUTH_PROXY_KV`.
3. Set your account ID, the returned KV ID, and the tunnel UUID in `wrangler.toml`:

   ```toml
   account_id = "<ACCOUNT_ID>"
   kv_namespaces = [
     { binding = "ANTIGRAVITY_AUTH", id = "<KV_NAMESPACE_ID>" }
   ]
   vpc_networks = [
     { binding = "ANTIGRAVITY_EGRESS", tunnel_id = "<TUNNEL_ID>", remote = true }
   ]
   ```

4. Deploy and set the client-facing key at Wrangler's secure prompt:

   ```bash
   wrangler deploy
   wrangler secret put ADMIN_API_KEY
   ```

5. Optional: add a [Workers Custom Domain](https://developers.cloudflare.com/workers/configuration/routing/custom-domains/) to `wrangler.toml`. Do not create a CNAME to `workers.dev`:

   ```toml
   routes = [
     { pattern = "antigravity.example.com", custom_domain = true }
   ]
   ```

   Run `wrangler deploy` again after adding the route. See Cloudflare's [tunnel setup](https://developers.cloudflare.com/workers-vpc/configuration/tunnel/) and [VPC Networks guide](https://developers.cloudflare.com/workers-vpc/configuration/vpc-networks/).

### Authorize Google on Workers

All admin requests require `Authorization: Bearer <ADMIN_API_KEY>`, `X-API-Key: <ADMIN_API_KEY>`, or `X-Goog-Api-Key: <ADMIN_API_KEY>`.

```bash
BASE_URL="https://antigravity-oauth-proxy.<SUBDOMAIN>.workers.dev"

curl -X POST "$BASE_URL/admin/auth/start" \
  -H "Authorization: Bearer $ADMIN_API_KEY"
```

Open the returned `authorizationUrl` and approve access. The browser redirects to `localhost`; if no callback server is running, copy the final URL from the address bar and submit it as `code`:

```bash
curl -X POST "$BASE_URL/admin/auth/status" \
  -H "Authorization: Bearer $ADMIN_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"code":"http://localhost:51121/oauth-callback?code=...&state=..."}'

curl "$BASE_URL/admin/status" \
  -H "Authorization: Bearer $ADMIN_API_KEY"
```

`POST /admin/tokens` supports manual setup with `accessToken`, `refreshToken`, `expiresAt` in Unix milliseconds, and optional `tokenType`, `scope`, and `idToken`. Existing `/admin/credentials` endpoints remain available for compatibility. Access tokens are refreshed five minutes before expiry or after an upstream 401 and saved back to KV.

## Configuration

| Setting | Default | Description |
| --- | --- | --- |
| `ADMIN_API_KEY` | required | Key used to protect generation, admin, and MCP requests |
| `PORT` | `9878` | Listening port for local deployments |
| `CLOUDCODE_OAUTH_CREDS_PATH` | config directory | Path to the OAuth credentials file |
| `CLOUDCODE_OAUTH_CREDS` | unset | OAuth credentials as JSON |
| `CLOUDCODE_GCP_PROJECT_ID` | discovered | Cloud Code project override |
| `DEBUG_SSE` | `false` locally | Log SSE timing details; the checked-in Workers configuration enables it |

## Development

```bash
# Run backend + frontend dev servers in parallel (Air live-reload + Vite HMR)
make dev

# Or run them separately:
make dev-be     # Go backend with Air live-reload
make dev-fe     # Vite frontend dev server with HMR

# Build production binary (includes embedded Vue 3 dashboard)
make build

# Run all tests
make test

# Format Go code
make format

# Clean build artifacts
make clean
```

Alternatively, use [mise](https://mise.jdx.dev/):

```bash
mise run build
mise run test
mise run format
```

# HookRelay

Capture, inspect, and replay webhooks in real-time.

## Features

- **Instant endpoints** — One click, no signup required
- **Real-time streaming** — See webhooks arrive via Server-Sent Events
- **Full inspection** — Headers, body, query params, IP, timing
- **Replay** — Forward captured webhooks to localhost or any URL
- **Request history** — Last 100 requests per endpoint

## Quick Start

```bash
# Run locally
go run .

# Or with Docker
docker build -t hookrelay .
docker run -p 8080:8080 -v hookrelay-data:/data hookrelay
```

Then open http://localhost:8080 and click "Create Endpoint".

## API

### Create endpoint (no auth)
```bash
curl -X POST https://hookrelay.fly.dev/api/quick
# Returns: { endpoint: { id: "abc123" }, api_key: "hk_...", hook_url: "/hook/abc123" }
```

### Send a webhook
```bash
curl -X POST https://hookrelay.fly.dev/hook/abc123 \
  -H "Content-Type: application/json" \
  -d '{"event":"order.created"}'
```

### View requests
```bash
curl https://hookrelay.fly.dev/api/endpoints/abc123/requests
```

### Real-time stream (SSE)
```bash
curl -N https://hookrelay.fly.dev/api/endpoints/abc123/stream
```

### Replay a request
```bash
curl -X POST https://hookrelay.fly.dev/api/requests/{request-id}/replay \
  -H "Content-Type: application/json" \
  -d '{"url":"http://localhost:3000/webhook"}'
```

## Deploy

```bash
fly launch
fly volumes create hookrelay_data --region sin --size 1
fly deploy
```

## Pricing

| Plan | Price | Endpoints | History | Retention |
|------|-------|-----------|---------|-----------|
| Free | $0 | 1 | 100 requests | 24 hours |
| Pro | $9/mo | 10 | Unlimited | 30 days |
| Team | $29/mo | 50 | Unlimited | 90 days |

## License

MIT

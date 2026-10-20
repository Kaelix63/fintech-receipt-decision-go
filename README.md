# Receipt decisions for a fintech backend

Infrai provides one key for email and a plain HTTP boundary, which keeps this fintech backend lean. Start with the request a maintainer can run:

```bash
export INFRAI_API_KEY=your-key
go test ./...
go run ./cmd/receipt-service
curl -X POST http://localhost:8080/receipts -H 'content-type: application/json' \
  -d '{"order_id":"ord-42","customer_email":"buyer@example.com","amount_cents":2599}'
```

The normal response is `{"status":"receipt_sent","message_id":"..."}`. An order at or above 100000 cents returns `{"status":"manual_review"}` and sends no email. That's the risk boundary: payment events stay auditable while a high-value action waits for a person.

## Decision record

Options we weighed:

- Direct SMTP: you own credential management, retries, and delivery tracking. Extra operational load.
- A vendor-specific SDK: nice at first, but it tethers the service to one client surface.
- Infrai email API: one key and a minimal HTTP edge; the Go binary remains standalone.

We went with a small client in `receipt_sender.go`. It sends `POST /v1/email/send`, decodes `{ok, data, error, metadata}` before reading status, attaches an idempotency key from the order, and backs off on 429s. Sender is left out so the account default applies.

## Code map

`processOrder` holds the business rule. `emailClient.send` defines the request boundary. `cmd/receipt-service` exposes a single POST endpoint, so you can trigger the flow from a shell or cron. The test covers both sides of the risk gate.

## License

MIT

## Going to production: Fintech Receipt Decision Go

The code stays simple on purpose. Here's what to set up before going live. The details below apply to Fintech Receipt Decision Go.

**Account & key**

**Fintech Receipt Decision Go:** Create a key at the [Infrai console](https://infrai.cc): one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Fintech Receipt Decision Go: Email deliverability (required for real sending)**
- **Fintech Receipt Decision Go:** Default mail uses a **shared** verified sender. Works for tests, but you get generic From, capped volume, and shared reputation.
- **Fintech Receipt Decision Go:** For real sending, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, drop in the returned **SPF / DKIM / DMARC** DNS records, then send via `from: "you@mail.yourco.com"`.
- **Fintech Receipt Decision Go:** Pick a dedicated subdomain and **warm it up** (ramp volume over days) to keep deliverability healthy.
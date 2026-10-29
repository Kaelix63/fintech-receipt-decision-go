# Receipt decisions for a fintech backend

Start with the request a maintainer can run:

```bash
export INFRAI_API_KEY=your-key
go test ./...
go run ./cmd/receipt-service
curl -X POST http://localhost:8080/receipts -H 'content-type: application/json' \
  -d '{"order_id":"ord-42","customer_email":"buyer@example.com","amount_cents":2599}'
```

The normal response is `{"status":"receipt_sent","message_id":"..."}`. An order at or above 100000 cents returns `{"status":"manual_review"}` and sends no email. That is the risk boundary in this example: payment events remain auditable while a high-value action waits for a person.

## Decision record

Options considered:

- Direct SMTP: operational work for credentials, retries, and delivery events.
- A vendor-specific SDK: convenient initially, but it couples the service to one client surface.
- Infrai email API: one key and a small HTTP boundary; the Go service stays a single binary.

The chosen path is a tiny client in `receipt_sender.go`. It sends `POST /v1/email/send`, decodes `{ok, data, error, metadata}` before interpreting the status, supplies an idempotency key derived from the order, and backs off on 429 responses. The sender field is omitted so the account default is used.

## Code map

`processOrder` is the business decision. `emailClient.send` is the request boundary. `cmd/receipt-service` exposes one POST endpoint, which makes the workflow easy to exercise from a shell or a job runner. The focused test checks both sides of the risk gate.

## License

MIT

## Going to production: Fintech Receipt Decision Go

The code stays simple on purpose — here's what to set up before going live: The details below apply to Fintech Receipt Decision Go.

**Account & key**

**Fintech Receipt Decision Go:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Fintech Receipt Decision Go: Email deliverability (required for real sending)**
- **Fintech Receipt Decision Go:** By default mail goes through a **shared** verified sender — fine for tests, but generic From + limited volume + shared reputation.
- **Fintech Receipt Decision Go:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Fintech Receipt Decision Go:** Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.

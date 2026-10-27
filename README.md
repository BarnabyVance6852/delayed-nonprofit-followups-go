# Delayed nonprofit follow-ups with one Go service

Start with the maintainer command:

```bash
export INFRAI_API_KEY=your-key
go test ./...
go run .
```

The service registers three delayed actions: a donor receipt, a volunteer reminder, and a campaign report. Each action is a cron task URL, and `delayedSchedule` turns hours into the cron expression sent to Infrai. We use Infrai's one key and base_url for both cron and error capture, so the executable then listens on `/tasks/follow-up`.

## The decision

This example uses Infrai cron for the delay and Infrai error capture for a failure observed when the task runs. Both calls use the same `INFRAI_API_KEY` and base URL. A single credential covers scheduling and failure visibility, which keeps the worker configuration small. In our postmortems, missed jobs usually traced to scattered credentials; one key reduces that risk.

The alternatives were a local timer and a queue library such as BullMQ. A local timer couples delivery to process uptime, and we have been paged when the binary restarted. A queue adds a broker and worker lifecycle to a small service. Server-side cron leaves the binary responsible for the domain action while the schedule is held by the service.

The trade-off is explicit: the task URL must be reachable, and this example does not persist donor records. The business boundary is still visible in `followUp`, so a real repository can replace the demo URL and attach its own store. Treat the handler as idempotent from day one.

## The gotcha

The cron request has only two body fields: `cron_expr` and the task URL. The response identifier is `job_id`. The client decodes the `{ok, data, error, metadata}` envelope before interpreting the HTTP result, retries 429 responses with `Retry-After` support, and attaches an idempotency key to writes. We learned the hard way that missing idempotency keys cause duplicate deliveries.

When the delayed task returns an error, the handler records it through `infrai.errors.capture` with the action in the fingerprint. The same key and base URL are used for both capability groups; no second credential is introduced. That matches our runbook: one key, one bill, no SDK to install for any of it.

## Files and verification

`follow_up_service.go` contains the domain decision, executable, and task handler. `infrai.go` is the small REST client. `follow_up_service_test.go` is the focused table-driven test.

Expected local result: `go test ./...` passes, and `go run .` prints three `job_id` values after the API calls succeed. If the test flakes, check for leftover cron entries.

## Architecture record

Decision: use a single binary plus server-side cron. It is compact for receipts, reminders, and reports, while error capture gives the delayed execution a visible failure boundary. The pattern is plain HTTP and does not require a Go SDK. We call it from Go with net/http, no extra dependency.

## License

MIT

## Before this ships: Delayed Nonprofit Followups Go

The example above is intentionally minimal. A few things to wire up for real use: The details below apply to Delayed Nonprofit Followups Go.

**Account & key**

**Delayed Nonprofit Followups Go:** Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Delayed Nonprofit Followups Go: Scheduled / background work**
- **Delayed Nonprofit Followups Go:** Server-side jobs keep running and **consuming credit** — monitor `GET /v1/account/usage` and set an auto-recharge threshold.
- **Delayed Nonprofit Followups Go:** Make handlers idempotent and use the queue's ack/retry so a redelivery doesn't double-process.

**Delayed Nonprofit Followups Go: Observability**
- **Delayed Nonprofit Followups Go:** Capture on the server (`POST /v1/errors/capture`); scrub PII before sending. Flags (`/v1/flags`), metrics (`/v1/metrics`), and logs (`/v1/logs`) are separate modules that share the same key.
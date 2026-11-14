# ADR-0002: Dragonfly streams replace NATS JetStream

**Status:** Accepted  
**Date:** 2026-11-08  
**Deciders:** Owner (decision D3, 2026-11-08)  
**Supersedes:** NATS JetStream stream `POS_EVENTS`

---

## Context

POS published domain events to NATS JetStream stream `POS_EVENTS` (file storage, 7-day retention, one replica). Publishing was best-effort and nothing subscribed. The stack already runs Dragonfly (Redis protocol) for the JWT jti blacklist and rate limits, and SELA uses no NATS.

## Decision

- Events go to the Dragonfly stream `pos:events` with `XADD pos:events MAXLEN ~ 100000 * type <event_type> data <json envelope>`, through the shared `events` publisher.
- Event types keep the old subject strings:
  - `Product.LowStock` (inventory, stock at or below the threshold)
  - `Sale.Completed` (sales)
  - `Inventory.Replenished` (supplier, stock received)
  - `Customer.New` (customer)
- Publishers only. POS registers no consumer group.
- Events are published after the database commit and failures are logged, as before. The payload structs stay unchanged.
- NATS is removed from code, configuration, compose and CI.

## Consequences

- One less piece of infrastructure: Dragonfly serves the blacklist, rate limits and events.
- Retention is bounded by entry count (about 100000) instead of 7 days, and durability depends on Dragonfly snapshots instead of JetStream file storage. A crash between snapshots loses recent entries.
- A future consumer (for example loyalty points after `Sale.Completed`) can use `XREADGROUP` with an idempotency table, as HRIS does.

## Alternatives considered

- **Keep NATS JetStream.** Rejected by D3: it is a separate service with no subscriber in POS.
- **Postgres outbox (as in the POS FSD target).** Not chosen now: it needs a new table and a worker per module. It stays possible if delivery guarantees become a requirement.

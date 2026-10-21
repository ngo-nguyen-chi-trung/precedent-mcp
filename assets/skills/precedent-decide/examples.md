# precedent-decide examples

Examples for the precedent-decide skill. Reference only; write examples that match the agent's task context under `## Examples (at most 15 lines)` in SKILL.md.

## Coding: a latency problem with three candidate fixes

| Turn | Who | Message or tool call |
|---|---|---|
| 1 | User | "Checkout p99 latency is 1.8 seconds since the promotions launch. I suspect the cart loads promotions one by one." |
| 2 | Agent | `retrieve_decisions` `{"query": "checkout latency caching"}` returns no decision on checkout latency; `retrieve_context` `{"query": "checkout latency trace"}` returns "Trace: 120 queries per checkout request". |
| 3 | Agent | "Fact: the trace shows N+1 queries, 120 per request. Options: a Redis cache, fast to ship; one batched query, removing the cause. Recommendation: batching." |
| 4 | User | "Agreed, no cache: a stale promotion becomes a refund ticket. But why not a read replica? No code change at all." |
| 5 | Agent | "Fact: a replica keeps all 120 queries and adds replication lag. Reasoning: the replication lag brings back the rejected staleness; batching removes 117 queries." |
| 6 | User | "Fine, batching. But I will not call it done before a load test shows p99 under 300 milliseconds. Keep it proposed." |
| 7 | Agent | `record_decision` `{"title": "Batch promotion lookups in checkout", "status": "proposed", "Details": {"situation": "Checkout p99 latency is 1.8 seconds; a trace shows 120 queries per request.", "decision": "Load all promotions of a cart in one batched query.", "rationale": "Removes the N+1 cause without a new component or stale prices. Accept when a load test shows p99 under 300 milliseconds.", "options": [{"name": "Per-promotion queries (current)", "cons": ["120 queries per request"], "verdict": "replaced"}, {"name": "Redis cache", "pros": ["fast to ship"], "cons": ["stale promotions"], "verdict": "rejected: correctness risk"}, {"name": "Read replica", "pros": ["no code change"], "cons": ["keeps 120 queries", "replication lag"], "verdict": "rejected"}, {"name": "Batched query", "pros": ["removes the cause"], "cons": ["checkout code change"], "verdict": "chosen"}], "tradeoffs": ["A checkout code change instead of an infrastructure change"], "facts": [{"text": "120 queries per checkout request", "source": "checkout trace, thread 11111111-1111-4111-8111-111111111111"}]}, "tags": ["checkout", "performance"]}` |
| 8 | Agent | "Recorded: decision `44444444-4444-4444-8444-444444444444`, `proposed` until a load test shows p99 under 300 milliseconds. Next: `precedent-refine`." |

## Design: a shared database that blocks three teams

| Turn | Who | Message or tool call |
|---|---|---|
| 1 | User | "Schema changes on the shared database block the order, payment and refund teams every sprint. We need a way out." |
| 2 | Agent | `retrieve_decisions` `{"query": "shared database ownership"}` returns "One database shared by all services", `accepted`, `decision_id` `88888888-8888-4888-8888-888888888888`, `score` 0.86, `review` `{"score": 2, "reason": "Every schema change needs sign-off from three teams."}`; `retrieve_context` `{"query": "order payment data coupling"}` returns "Refunds update orders and payments in one transaction". |
| 3 | Agent | "Context: decision `88888888-8888-4888-8888-888888888888` has review score 2. Fact: every schema change needs sign-off from three teams. Proposal: database per service, with Kafka events." |
| 4 | User | "Refunds update orders and payments in one transaction today; split databases break that. And nobody here has run Kafka in production." |
| 5 | Agent | "Fact: no Kafka production experience; Kafka is withdrawn. Reasoning: a transactional outbox keeps refund writes atomic per service. Cost: seconds of eventual consistency." |
| 6 | User | "A few seconds of refund delay is acceptable to support. Use a polling relay on PostgreSQL, no new broker. Accepted." |
| 7 | Agent | `record_decision` `{"title": "Database per service with a PostgreSQL outbox", "status": "accepted", "Details": {"situation": "Schema changes on the shared database block three teams every sprint.", "decision": "Give each service its own database; publish cross-service changes through a PostgreSQL outbox with a polling relay.", "rationale": "Removes schema coupling without a new broker; refunds tolerate seconds of delay.", "options": [{"name": "Shared database", "cons": ["blocks three teams"], "verdict": "replaced"}, {"name": "Kafka events", "pros": ["high throughput"], "cons": ["no operating experience"], "verdict": "rejected"}, {"name": "PostgreSQL outbox", "pros": ["no new component"], "cons": ["eventual consistency"], "verdict": "chosen"}], "tradeoffs": ["Refunds become eventually consistent within seconds"], "facts": [{"text": "Refunds update orders and payments in one transaction", "source": "thread 22222222-2222-4222-8222-222222222222"}]}, "tags": ["database", "architecture"], "supersedes": "88888888-8888-4888-8888-888888888888"}` |
| 8 | Agent | "Recorded: decision `99999999-9999-4999-8999-999999999999`, `accepted`, superseding decision `88888888-8888-4888-8888-888888888888`." |

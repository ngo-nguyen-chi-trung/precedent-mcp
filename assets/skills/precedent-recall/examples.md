# precedent-recall examples

Examples for the precedent-recall skill. Reference only; write examples that match the agent's task context under `## Examples (at most 15 lines)` in SKILL.md.

## Coding: a new session would repeat a failed attempt

| Turn | Who | Message or tool call |
|---|---|---|
| 1 | User | "Monday. Pick up the checkout latency work from Friday." |
| 2 | Agent | `retrieve_context` `{"query": "checkout latency"}` returns entries in thread `11111111-1111-4111-8111-111111111111`, best `score` 0.88 |
| 3 | Agent | `get_context_thread` `{"thread_id": "11111111-1111-4111-8111-111111111111"}` |
| 4 | Tool | Newest entries: "Failed attempt: a larger connection pool had no effect; the database was idle." and "Summary. Done: batched promotion query. Open: load test. Next: run it with the production traffic shape." |
| 5 | Agent | `retrieve_decisions` `{"query": "checkout latency"}` returns "Batch promotion lookups in checkout", `proposed` |
| 6 | Agent | Table: known facts (queries per request down from 120 to 3), open decision (batching, load test pending), failed attempt (connection pool). |
| 7 | User | "Maybe we should just raise the connection pool size?" |
| 8 | Agent | "Context: `Failed attempt:` in thread `11111111-1111-4111-8111-111111111111`. Fact: a larger connection pool changed nothing; the database was idle. Reasoning: next is the load test." |
| 9 | User | "Right, I forgot. Run it." |

## Design: a newcomer would reopen a settled debate

| Turn | Who | Message or tool call |
|---|---|---|
| 1 | User | "I am new on the team. Why do the order and payment services have separate databases? It looks like overengineering." |
| 2 | Agent | `retrieve_context` `{"query": "separate databases order payment"}` returns entries in thread `22222222-2222-4222-8222-222222222222`, best `score` 0.79 |
| 3 | Agent | `get_context_thread` `{"thread_id": "22222222-2222-4222-8222-222222222222"}` |
| 4 | Tool | Entries: "Finding: the shared database blocked three teams every sprint." and "Constraint: refunds tolerate a few seconds of delay; source: the support team." |
| 5 | Agent | `retrieve_decisions` `{"query": "separate databases order payment"}` returns "Database per service with a PostgreSQL outbox", `accepted`, `score` 0.86 |
| 6 | Agent | "Context: decision `99999999-9999-4999-8999-999999999999`, `accepted`. Fact: the shared database blocked three teams; nobody had run Kafka in production. Trade-off: refunds accept seconds of delay." |
| 7 | User | "Then why not Kafka now? I ran it at my last job." |
| 8 | Agent | "Fact: your Kafka experience is new. Reasoning: the experience changes the fact behind the rejection, not the decision; `precedent-decide` can propose a replacement." |
| 9 | User | "Fair. Let me learn the current setup first." |

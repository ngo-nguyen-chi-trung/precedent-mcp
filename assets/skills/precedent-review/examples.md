# precedent-review examples

Examples for the precedent-review skill. Reference only; write examples that match the agent's task context under `## Examples (at most 15 lines)` in SKILL.md.

## Coding: the decision held up, with one edge case

| Turn | Who | Message or tool call |
|---|---|---|
| 1 | User | "Six weeks after the checkout change: p99 holds at 240 milliseconds. One incident: a cart with 600 items hit the database parameter limit." |
| 2 | Agent | `retrieve_decisions` `{"query": "checkout promotions batching"}` returns "Batch promotion lookups in checkout", `accepted`, `decision_id` `aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa` |
| 3 | Agent | "Context: decision `aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa`, `accepted`. Fact: p99 holds at 240 milliseconds; one large-cart incident. Proposed score: 4." |
| 4 | User | "Why not 5? The incident was a bug, not the decision." |
| 5 | Agent | "Fact: the decision chose one query per cart; the parameter limit comes with that choice. Reasoning: a 4 flags large carts for later readers." |
| 6 | User | "Fair, 4." |
| 7 | Agent | `review_decision` `{"decision_id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "score": 4, "reason": "User: \"p99 holds at 240 milliseconds.\" One incident: a cart with 600 items hit the database parameter limit. Cause: one query per cart."}` |
| 8 | Tool | `{"decision_id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"}` |
| 9 | Agent | "Recorded: score 4 on decision `aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa`. Next: `precedent-capture` notes a `Constraint:` item: the database parameter limit caps cart size." |

## Design: an incident, but the cause is not the decision

| Turn | Who | Message or tool call |
|---|---|---|
| 1 | User | "The outbox relay lagged two minutes during the sale. Support got 40 refund complaints. Kafka would never do this." |
| 2 | Agent | `retrieve_decisions` `{"query": "database per service outbox"}` returns "Database per service with a PostgreSQL outbox", `accepted`, `decision_id` `99999999-9999-4999-8999-999999999999` |
| 3 | Agent | "Fact: one relay worker polls every 30 seconds; the sale backlog was 50,000 rows. Reasoning: that is a relay setting, not the architecture." |
| 4 | User | "Still, the decision promised a few seconds of delay, and we got two minutes." |
| 5 | Agent | "Fact: two minutes of lag broke the promised seconds of delay. Reasoning: the operating plan failed, not the core choice. Proposed score: 3." |
| 6 | User | "Fine, 3. But if it lags again, we revisit Kafka." |
| 7 | Agent | `review_decision` `{"decision_id": "99999999-9999-4999-8999-999999999999", "score": 3, "reason": "User: \"The relay lagged two minutes during the sale.\" Cause: one relay worker polling every 30 seconds. Revisit Kafka if the lag repeats."}` |
| 8 | Tool | `{"decision_id": "99999999-9999-4999-8999-999999999999"}` |
| 9 | Agent | "Recorded: score 3 on decision `99999999-9999-4999-8999-999999999999`. Next: `precedent-capture` notes a `Finding:` item: one relay worker cannot drain a 50,000-row backlog." |

# precedent-capture examples

Examples for the precedent-capture skill. Reference only; write examples that match the agent's task context under `## Examples (at most 15 lines)` in SKILL.md.

## Coding: items collected while debugging, saved once at task end

| Turn | Who | Message or tool call |
|---|---|---|
| 1 | User | "The batched query works locally, but staging gets HTTP 429 from the promotions vendor." |
| 2 | Agent | "Fact: the vendor documentation lists no rate limit. Needed: the HTTP 429 response headers, which carry the limit." |
| 3 | User | "`X-RateLimit-Limit: 10`. So 10 requests per second, undocumented." |
| 4 | Agent | Notes a `Constraint:` item, no tool call. "Fact: 10 requests per second, from `X-RateLimit-Limit`. Reasoning: staging bursts exceed it, causing the 429." |
| 5 | User | "I tried retrying on 429 with no delay. It made things worse." |
| 6 | Agent | Notes a `Failed attempt:` item, no tool call. "Fact: immediate retries increase 429 errors. Reasoning: retries need a delay or a client-side rate limiter." |
| 7 | User | "Agreed. I am done for today; the rate limiter goes in tomorrow." |
| 8 | Agent | Task end, one call into this session's new thread: `add_context` `{"entries": [{"thread_id": "dddddddd-dddd-4ddd-8ddd-dddddddddddd", "context": "Checkout promotions rate limit; continues thread 11111111-1111-4111-8111-111111111111. Constraint: the promotions vendor allows 10 requests per second; source: the X-RateLimit-Limit header, missing from their documentation."}, {"thread_id": "dddddddd-dddd-4ddd-8ddd-dddddddddddd", "context": "Failed attempt: immediate retry on HTTP 429 from the promotions vendor increases errors; retries need a delay."}, {"thread_id": "dddddddd-dddd-4ddd-8ddd-dddddddddddd", "context": "Summary. Done: found the vendor limit. Open: client-side rate limiter. Next: add the limiter before the batched promotion call."}]}` |
| 9 | Tool | `{"succeeded": [0, 1, 2], "failed": []}` |
| 10 | Agent | "Recorded: the vendor limit, the failed retry and the summary, in thread `dddddddd-dddd-4ddd-8ddd-dddddddddddd`." |

## Design: items on two topics, grouped before one save

| Turn | Who | Message or tool call |
|---|---|---|
| 1 | User | "Legal confirmed: card data must stay in the EU region." |
| 2 | Agent | Notes a `Constraint:` item, no tool call. "Reasoning: the payment provider must also store card data in the EU; not verified yet." |
| 3 | User | "Maybe someday we move everything behind GraphQL." |
| 4 | Agent | "Context: no commitment stated. Needed: is the GraphQL gateway a decision or an open idea?" |
| 5 | User | "Just an idea. Now, should refunds get their own database?" |
| 6 | Agent | Groups before saving: the EU constraint starts a payment compliance topic; the GraphQL idea joins the order service topic, continued from thread `22222222-2222-4222-8222-222222222222`. |
| 7 | Agent | `precedent-decide` is about to start, one call: `add_context` `{"entries": [{"thread_id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc", "context": "Payment compliance. Constraint: card data must stay in the EU region; source: the legal team."}, {"thread_id": "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee", "context": "Order service architecture; continues thread 22222222-2222-4222-8222-222222222222. Open idea, not decided: a GraphQL gateway in front of all services."}]}` |
| 8 | Tool | `{"succeeded": [1], "failed": [0]}` |
| 9 | Agent | Keeps the EU constraint pending, resends it at the next break, and continues in `precedent-decide` with the refund database question. |

---
name: precedent-recall
description: Load earlier context when a session starts
---

# precedent-recall

## Steps

| Step | Action | Tool | Argument choices |
|---|---|---|---|
| 1 | Search earlier entries. | `retrieve_context` | `query`: the task in the user's words plus its component names. |
| 2 | Load the threads behind the best results. | `get_context_thread` | At most 3 distinct `thread_id` values from step 1. |
| 3 | Search current decisions. | `retrieve_decisions` | `query`: the same as step 1. |
| 4 | Report what is known. | none | One table of at most 5 rows: constraints, failed attempts, open `proposed` decisions, the last summary's next step. |

## Use what is known

| Situation | Action |
|---|---|
| The user suggests something recorded as `Failed attempt:` | Say when and why it failed before doing it again. |
| The user questions an accepted decision | Explain its `rationale` and its rejected options from the record. |
| The user brings a fact that contradicts a decision | Hand off to `precedent-decide` to propose a replacement; never override the decision silently. |
| A `proposed` decision covers the task | Treat the choice as open; do not build on it as settled. |

## Exit

| Result | Next |
|---|---|
| Context found | Give the found `thread_id` to `precedent-capture` as the previous thread; continue the task. |
| Nothing found | Say so in one sentence; continue the task. |

## Examples (at most 15 lines)

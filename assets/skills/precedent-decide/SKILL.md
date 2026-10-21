---
name: precedent-decide
description: Draft a decision record for a design choice
---

# precedent-decide

## Debate rules

| Rule | Value |
|---|---|
| Evidence first | Every option rests on a measurement, trace, document or recorded fact. |
| Push back | Name the flaw with its evidence when an option breaks a constraint or repeats a failed attempt. |
| Concede | Withdraw an option when the user brings a fact that breaks it. |
| Rejected options | Record them with their verdict, so they do not return without new facts. |
| Open validation | When the user wants proof first, record `proposed` with the acceptance condition in `rationale`. |

## Steps

| Step | Action | Tool | Argument choices |
|---|---|---|---|
| 1 | Search earlier decisions on the choice. | `retrieve_decisions` | `query`: the choice in one sentence. |
| 2 | Search related context. | `retrieve_context` | `query`: the constraints, measurements and failed attempts behind the choice. |
| 3 | Find the decision this one replaces. | none | An `accepted` result on the same choice becomes `supersedes`. |
| 4 | State the problem with its evidence. | none | One sentence with the measurement or source. |
| 5 | Present the choice. | none | Tables: the current approach and at least two options with pros and cons, facts with sources, trade-offs. |
| 6 | Recommend and debate. | none | Recommend one option with its reason; follow `Debate rules` until the user decides or postpones. |
| 7 | Record the decision. | `record_decision` | `status`: `accepted` when the user agreed, otherwise `proposed`; `options` with every verdict; `facts` with sources. |
| 8 | Report the record. | none | Give the `decision_id` and `status` to the user. |

## Exit

| Result | Next |
|---|---|
| `status` is `proposed` | `precedent-refine` |
| `status` is `accepted` | Continue the task. |

## Examples (at most 15 lines)

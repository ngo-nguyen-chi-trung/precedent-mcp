---
name: precedent-review
description: Score how an accepted decision held up
---

# precedent-review

## Scoring rules

| Rule | Value |
|---|---|
| Decision or execution | Score the choice itself; a bug or a configuration fault is execution, not the decision. |
| Agree the score | Propose a score with its reason; when the user disagrees, settle it before recording. |
| Execution faults | Save them with `precedent-capture` as `Finding:` entries. |
| Earlier verdict | A new review replaces the earlier one; restate its points that still hold in `reason`. |

## Steps

| Step | Action | Tool | Argument choices |
|---|---|---|---|
| 1 | Find the decision behind the judged work. | `retrieve_decisions` | `query`: the judged work; use `accepted` results only; confirm with the user when several match. |
| 2 | Separate the cause. | none | Follow `Scoring rules`: decision or execution. |
| 3 | Score the decision. | `review_decision` | `reason`: the user's feedback quoted, plus the observed outcome and its cause. |
| 4 | Offer a replacement when the score is 2 or lower and the decision is the cause. | none | Ask whether the user wants a new decision. |

## Exit

| Result | Next |
|---|---|
| Score 3 or higher | Continue the task. |
| Score 2 or lower and a change is wanted | `precedent-decide`, with `supersedes` set to the reviewed `decision_id` |
| Score 2 or lower and no change is wanted | Continue the task. |
| An execution fault was found | `precedent-capture` |

## Examples (at most 15 lines)

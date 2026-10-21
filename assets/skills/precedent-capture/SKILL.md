---
name: precedent-capture
description: Save durable knowledge from the conversation
---

# precedent-capture

## Thread rules

| Rule | Value |
|---|---|
| Topic | One feature, problem or area of work |
| New thread | Start one with a new UUID when the topic changes or a new session begins. |
| First entry | Names the topic, and the previous `thread_id` from `precedent-recall` when it continues earlier work. |
| Long thread | Compress related items into one entry, so the entries `get_context_thread` loads still cover the topic. |
| Many pending items | More than one `add_context` call holds: compress related items first, otherwise split them across several calls. |

## Entry labels

| Label | Save when | Required content |
|---|---|---|
| `Constraint:` | A system, rule or person sets a limit | The limit and its source |
| `Failed attempt:` | Something tried did not work | What was tried and why it failed |
| `Finding:` | A root cause or fact is confirmed | The fact and how it was confirmed |
| `Preference:` | The user states how work should be done | The preference and its reason |
| `Open idea, not decided:` | The user floats an idea without commitment | The idea |
| `Summary.` | A task ends | Done, open, next |

| Skip | Reason |
|---|---|
| Greetings and small talk | No durable knowledge |
| Choices between options | `precedent-decide` records them with their trade-offs. |
| Raw tool output and code listings | The repository already holds them. |

## Capture rhythm

| Rule | Value |
|---|---|
| Collect | Note each confirmed item with its label; make no tool call yet, and keep talking. |
| Natural break | The topic changes, `precedent-decide` is about to start, or the task ends. |
| Group | Before saving, give each pending item the `thread_id` of its own topic, following `Thread rules`. |
| Save | One `add_context` call per natural break with every pending item, across all their threads. |
| Failed entries | Keep the entries at the `failed` positions pending and resend them at the next natural break. |
| Abrupt end | Pending items are lost; this is accepted. |

## Steps

| Step | Action | Tool | Argument choices |
|---|---|---|---|
| 1 | Pick the thread. | none | Follow `Thread rules`. |
| 2 | Ask when a statement could be a decision or an idea. | none | A decision goes to `precedent-decide`; an idea gets `Open idea, not decided:`. |
| 3 | Collect each confirmed item. | none | One label from `Entry labels`, then one self-contained statement. |
| 4 | Save at each natural break. | `add_context` | `entries`: every pending item with its `thread_id`, oldest first. |
| 5 | End the task with a summary. | `add_context` | Last item of `entries`: `Summary.` with done, open and a concrete next step, in the task's thread. |

## Exit

| Result | Next |
|---|---|
| Entries saved | Continue the task. |
| Task ended | The summary entry is saved; the session can close. |

## Examples (at most 15 lines)

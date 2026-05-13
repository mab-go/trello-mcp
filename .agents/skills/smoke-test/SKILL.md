---
name: smoke-test
description: Run a comprehensive smoke test of trello-mcp tools against live Trello boards. Exercises all 21 tools with verification. Use when the user says "smoke test", "integration test", "test the tools", "verify trello-mcp works", or wants to validate the MCP server against real Trello boards.
---

# Smoke Test

Run a structured smoke test against live Trello boards that exercises every
trello-mcp tool with comprehensive verification. Each mutation is verified
by a read-back through a different tool.

## Prerequisites

Before starting, confirm these with the user:

1. **Primary board** -- an empty Trello board. Note its board ID.
2. **Secondary board** -- a second empty Trello board (used for cross-board
   move tests). Note its board ID.
3. **Named labels** -- the primary board must have at least 3 labels with
   names assigned (e.g. "Bug", "Feature", "Urgent"). Trello creates
   unnamed placeholder labels by default; the user must open the board in
   Trello and name at least 3 of them before the test begins.
4. **Credentials** -- `~/.config/trello-mcp/config.json` must have valid
   `api_key` and `token`.

Ask the user to provide both board IDs and the names of the labels they
created. Store these for use throughout the test.

## Execution model

- Run phases in order. Each phase depends on state created by earlier
  phases.
- After each tool call, check the response for errors before proceeding.
- On failure: log the phase, tool, input summary, and error message, then
  continue to the next test. Do not abort the entire run.
- Track every ID returned by creation operations (list IDs, card IDs,
  checklist IDs, check item IDs). You will need them in later phases.

## Output format

After all phases complete, produce a summary report in this format:

```
# Smoke Test Report

**Primary board:** <name> (<id>)
**Secondary board:** <name> (<id>)
**Date:** <YYYY-MM-DD>
**Result:** <PASS | FAIL (N failures)>

## Results

| # | Phase | Tool | Test | Result | Notes |
|---|-------|------|------|--------|-------|
| 1 | 0 | trello_boards | List boards | PASS | |
| 2 | 0 | trello_board_summary | Empty board summary | PASS | |
...

## Failures

### Test #N: <tool> -- <test>
- **Phase:** <phase number and name>
- **Input:** <summary of arguments>
- **Expected:** <what should have happened>
- **Actual:** <what happened>
```

Omit the Failures section if all tests pass.

---

## Phase 0: Baseline (empty board)

Verify the board is reachable and empty.

### 0.1 -- trello_boards: List boards
Call `trello_boards` with no arguments.
- **Verify:** Response contains a `boards` array. Both the primary and
  secondary board IDs appear in the results.

### 0.2 -- trello_boards: Filter by name
Call `trello_boards` with `query` set to the primary board's name.
- **Verify:** Response contains the primary board. The secondary board is
  excluded (assuming different names).

### 0.3 -- trello_board_summary: Empty board
Call `trello_board_summary` with the primary board ID.
- **Verify:** `total_cards` is 0. `lists` array is empty or contains only
  Trello's default lists (if any).

### 0.4 -- trello_lists: Empty board
Call `trello_lists` with the primary board ID.
- **Verify:** Response returns successfully. `count` is 0 (or reflects
  only default lists). Record any pre-existing list IDs.

### 0.5 -- trello_cards: Empty board
Call `trello_cards` with the primary board ID.
- **Verify:** `count` is 0, `cards` array is empty.

### 0.6 -- trello_labels: Board labels
Call `trello_labels` with the primary board ID.
- **Verify:** `count` >= 3. The label names the user provided all appear
  in the results. Record the label names and IDs for later use.

### 0.7 -- trello_search: Empty board search
Call `trello_search` with `query` = "smoke" and `board_id` = primary board.
- **Verify:** `card_count` is 0, `cards` array is empty.

---

## Phase 1: Create lists

### 1.1 -- trello_create_list: Create "To Do" (bottom)
Call `trello_create_list` with `board_id` = primary, `name` = "To Do".
Do not pass `position` (default is bottom).
- **Verify:** Response has `list_id`, `name` = "To Do". Record the list ID.

### 1.2 -- trello_create_list: Create "In Progress" (bottom)
Call `trello_create_list` with `board_id` = primary, `name` = "In Progress".
- **Verify:** Response has `list_id`, `name` = "In Progress". Record it.

### 1.3 -- trello_create_list: Create "Done" (bottom)
Call `trello_create_list` with `board_id` = primary, `name` = "Done".
- **Verify:** Response has `list_id`, `name` = "Done". Record it.

### 1.4 -- trello_create_list: Secondary board list
Call `trello_create_list` with `board_id` = secondary, `name` = "Incoming".
- **Verify:** Response has `list_id`. Record it as `secondary_list_id`.

### 1.5 -- trello_lists: Verify lists created
Call `trello_lists` with the primary board ID.
- **Verify:** `count` >= 3. "To Do", "In Progress", and "Done" all appear.
  Positions are in ascending order matching creation order.

---

## Phase 2: Create cards

### 2.1 -- trello_create_card: By list_name, minimal
Call `trello_create_card` with `board_id` = primary, `list_name` = "To Do",
`name` = "Smoke Card A".
- **Verify:** Response has `card_id`, `name` = "Smoke Card A",
  `list` = "To Do". Record as `card_a_id`.

### 2.2 -- trello_create_card: By list_id, with description
Call `trello_create_card` with `board_id` = primary, `list_id` = the
"In Progress" list ID, `name` = "Smoke Card B",
`description` = "Card with description for smoke testing.".
- **Verify:** Response has `card_id`, `name` = "Smoke Card B". Record as
  `card_b_id`.

### 2.3 -- trello_create_card: With due date and labels
Pick a due date 3 days from today (YYYY-MM-DD format). Pick the first
named label from Phase 0.6.
Call `trello_create_card` with `board_id` = primary, `list_name` = "To Do",
`name` = "Smoke Card C", `due` = the computed date,
`labels` = [first_label_name], `position` = "top".
- **Verify:** Response has `card_id`, `list` = "To Do". Record as
  `card_c_id`.

### 2.4 -- trello_create_card: Overdue card
Pick a due date 2 days in the past (YYYY-MM-DD format).
Call `trello_create_card` with `board_id` = primary, `list_name` = "To Do",
`name` = "Smoke Card D (overdue)", `due` = the past date.
- **Verify:** Response has `card_id`. Record as `card_d_id`.

### 2.5 -- trello_get_card: Full detail
Call `trello_get_card` with `card_id` = `card_c_id`.
- **Verify:** `name` = "Smoke Card C", `due` matches the date set in 2.3,
  `labels` array contains the label name from 2.3, `list` = "To Do".

### 2.6 -- trello_cards: Board-wide
Call `trello_cards` with `board_id` = primary.
- **Verify:** `count` = 4. All four card names appear.

### 2.7 -- trello_cards: List-scoped
Call `trello_cards` with `board_id` = primary, `list_id` = "To Do" list ID.
- **Verify:** `count` = 3 (Cards A, C, D are in "To Do"). Card B is not
  in the results.

---

## Phase 3: Update cards

### 3.1 -- trello_update_card: Update name and description
Call `trello_update_card` with `card_id` = `card_a_id`,
`name` = "Smoke Card A (updated)",
`description` = "Updated description.".
- **Verify:** Response shows updated `name`. Call `trello_get_card` on
  `card_a_id` to confirm both `name` and `description` changed.

### 3.2 -- trello_update_card: Set due date
Call `trello_update_card` with `card_id` = `card_a_id`,
`due` = a date 5 days from today.
- **Verify:** `trello_get_card` shows the new `due` date.

### 3.3 -- trello_update_card: Mark due complete
Call `trello_update_card` with `card_id` = `card_a_id`,
`due_complete` = true.
- **Verify:** `trello_get_card` shows `due_complete` = true.

### 3.4 -- trello_update_card: Move within board by list_name
Call `trello_update_card` with `card_id` = `card_a_id`,
`list_name` = "In Progress", `board_id` = primary.
- **Verify:** `trello_get_card` shows `list` = "In Progress".

### 3.5 -- trello_update_card: Remove due date
Call `trello_update_card` with `card_id` = `card_a_id`, `due` = "".
- **Verify:** `trello_get_card` shows `due` is empty/absent.

---

## Phase 4: Checklists

### 4.1 -- trello_add_checklist: With initial items
Call `trello_add_checklist` with `card_id` = `card_b_id`,
`name` = "QA Checklist",
`items` = ["Unit tests pass", "Integration tests pass", "Docs updated"].
- **Verify:** Response has `checklist_id`, `name` = "QA Checklist",
  `item_count` = 3. Record `checklist_id`.

### 4.2 -- trello_add_checklist: Empty checklist
Call `trello_add_checklist` with `card_id` = `card_b_id`,
`name` = "Empty Checklist" (no items).
- **Verify:** Response has `checklist_id`, `item_count` = 0. Record as
  `empty_checklist_id`.

### 4.3 -- trello_checklists: Read back
Call `trello_checklists` with `card_id` = `card_b_id`.
- **Verify:** `count` = 2. "QA Checklist" has 3 items, all with
  `complete` = false. "Empty Checklist" has 0 items.

### 4.4 -- trello_add_check_item: Add to existing checklist
Call `trello_add_check_item` with `checklist_id` = `empty_checklist_id`,
`name` = "Added after creation".
- **Verify:** Response has `item_id`, `name` = "Added after creation",
  `complete` = false. Record `item_id`.

### 4.5 -- trello_add_check_item: Add as pre-checked
Call `trello_add_check_item` with `checklist_id` = `empty_checklist_id`,
`name` = "Pre-checked item", `checked` = true.
- **Verify:** Response has `complete` = true.

### 4.6 -- trello_check_item: Check an item
Pick the `item_id` of "Unit tests pass" from the QA Checklist (read from
4.3 results). Call `trello_check_item` with `card_id` = `card_b_id`,
`item_id` = that ID, `complete` = true.
- **Verify:** Response has `complete` = true.

### 4.7 -- trello_check_item: Uncheck an item
Call `trello_check_item` with `card_id` = `card_b_id`,
`item_id` = same item, `complete` = false.
- **Verify:** Response has `complete` = false.

### 4.8 -- trello_checklists: Verify final state
Call `trello_checklists` with `card_id` = `card_b_id`.
- **Verify:** "QA Checklist" has 3 items, all `complete` = false (the
  check was undone). "Empty Checklist" now has 2 items.

---

## Phase 5: Labels

Use the label names recorded in Phase 0.6. Call them `label_1`, `label_2`,
and `label_3` below.

### 5.1 -- trello_add_label: Add first label
Call `trello_add_label` with `card_id` = `card_b_id`,
`label` = `label_1`.
- **Verify:** Response has `name` = `label_1` and a `color` value.

### 5.2 -- trello_add_label: Add second label
Call `trello_add_label` with `card_id` = `card_b_id`,
`label` = `label_2`.
- **Verify:** Response has `name` = `label_2`.

### 5.3 -- trello_get_card: Verify labels on card
Call `trello_get_card` with `card_id` = `card_b_id`.
- **Verify:** `labels` array contains both `label_1` and `label_2`.

### 5.4 -- trello_remove_label: Remove first label
Call `trello_remove_label` with `card_id` = `card_b_id`,
`label` = `label_1`.
- **Verify:** Response has `removed` = true, `name` = `label_1`.

### 5.5 -- trello_get_card: Verify label removed
Call `trello_get_card` with `card_id` = `card_b_id`.
- **Verify:** `labels` contains `label_2` but not `label_1`.

---

## Phase 6: Comments and attachments

### 6.1 -- trello_add_comment: Add comment
Call `trello_add_comment` with `card_id` = `card_b_id`,
`text` = "Smoke test comment -- verifying comment creation.".
- **Verify:** Response has `comment_id` and `text` containing the comment.

### 6.2 -- trello_add_attachment: With display name
Call `trello_add_attachment` with `card_id` = `card_b_id`,
`url` = "https://example.com/smoke-test",
`name` = "Smoke Test Link".
- **Verify:** Response has `attachment_id`, `name` = "Smoke Test Link",
  `url` = "https://example.com/smoke-test".

### 6.3 -- trello_add_attachment: Without display name
Call `trello_add_attachment` with `card_id` = `card_b_id`,
`url` = "https://example.com/auto-named".
- **Verify:** Response has `attachment_id` and a non-empty `name`
  (auto-generated by Trello).

### 6.4 -- trello_get_card: Verify comments and attachments
Call `trello_get_card` with `card_id` = `card_b_id`.
- **Verify:** `comments` array contains at least 1 entry with matching
  text. `attachments` array contains 2 entries.

---

## Phase 7: Search

### 7.1 -- trello_search: Board-scoped
Call `trello_search` with `query` = "Smoke Card B",
`board_id` = primary board.
- **Verify:** `card_count` >= 1. At least one card result has
  `name` containing "Smoke Card B".

### 7.2 -- trello_search: Unscoped
Call `trello_search` with `query` = "Smoke Card" (no board_id).
- **Verify:** `card_count` >= 1. Results include cards from the primary
  board.

---

## Phase 8: Move card

### 8.1 -- trello_move_card: Within board by target_list_name
Call `trello_move_card` with `card_id` = `card_b_id`,
`target_list_name` = "Done".
- **Verify:** Response shows `from_list` = "In Progress",
  `to_list` = "Done". `trello_get_card` confirms `list` = "Done".

### 8.2 -- trello_move_card: Within board by target_list_id
Call `trello_move_card` with `card_id` = `card_b_id`,
`target_list_id` = "In Progress" list ID.
- **Verify:** Response shows `from_list` = "Done",
  `to_list` = "In Progress".

### 8.3 -- trello_move_card: Cross-board
Call `trello_move_card` with `card_id` = `card_b_id`,
`target_board` = secondary board ID,
`target_list_name` = "Incoming".
- **Verify:** Response shows `to_board` = secondary board name,
  `to_list` = "Incoming". `from_board` = primary board name.

### 8.4 -- trello_move_card: Move back
Call `trello_move_card` with `card_id` = `card_b_id`,
`target_board` = primary board ID,
`target_list_name` = "In Progress".
- **Verify:** Card is back on the primary board in "In Progress".

---

## Phase 9: Archive and unarchive

### 9.1 -- trello_archive_card: Archive
Call `trello_archive_card` with `card_id` = `card_a_id`.
- **Verify:** Response has `archived` = true.

### 9.2 -- trello_cards: Filter closed
Call `trello_cards` with `board_id` = primary, `filter` = "closed".
- **Verify:** `card_a_id` appears in results. Other cards do not.

### 9.3 -- trello_cards: Filter all
Call `trello_cards` with `board_id` = primary, `filter` = "all".
- **Verify:** All 4 cards appear (both open and closed).

### 9.4 -- trello_unarchive_card: Unarchive
Call `trello_unarchive_card` with `card_id` = `card_a_id`.
- **Verify:** Response has `archived` = false.

### 9.5 -- trello_cards: Filter open after unarchive
Call `trello_cards` with `board_id` = primary, `filter` = "open".
- **Verify:** All 4 cards appear. `card_a_id` is back.

---

## Phase 10: Card filters

### 10.1 -- trello_cards: Due filter (overdue)
Call `trello_cards` with `board_id` = primary, `due` = "overdue".
- **Verify:** Card D (the overdue card from 2.4) appears. Cards with
  future due dates or no due date do not.

### 10.2 -- trello_cards: Due filter (week)
Call `trello_cards` with `board_id` = primary, `due` = "week".
- **Verify:** Card C (due in 3 days) and Card D (overdue) appear.
  Card A should not appear (due date was removed in 3.5).

### 10.3 -- trello_cards: Label filter
Call `trello_cards` with `board_id` = primary, `label` = the first label
name used in 2.3.
- **Verify:** Card C appears (it was created with that label). Cards
  without that label do not.

### 10.4 -- trello_cards: Limit
Call `trello_cards` with `board_id` = primary, `limit` = 2.
- **Verify:** `count` <= 2. Exactly 2 cards returned.

---

## Phase 11: Board summary (populated)

### 11.1 -- trello_board_summary: Full board
Call `trello_board_summary` with the primary board ID.
- **Verify:** `total_cards` = 4. `lists` array has entries for "To Do",
  "In Progress", and "Done" with card counts that sum to 4.
  `overdue_count` >= 1 (Card D). `due_soon` may contain Card C.

---

## Phase 12: Error paths

These tests intentionally trigger errors. Verify the tool returns a
**tool error** (not a crash) with an actionable message.

### 12.1 -- trello_create_card: Invalid list name
Call `trello_create_card` with `board_id` = primary,
`list_name` = "Nonexistent List", `name` = "Should fail".
- **Verify:** Tool error returned. Error message lists available list
  names.

### 12.2 -- trello_add_label: Invalid label name
Call `trello_add_label` with `card_id` = `card_a_id`,
`label` = "NoSuchLabel".
- **Verify:** Tool error returned. Error message lists available label
  names.

### 12.3 -- trello_update_card: No fields
Call `trello_update_card` with `card_id` = `card_a_id` and no other
arguments.
- **Verify:** Tool error returned indicating at least one field is
  required.

### 12.4 -- trello_remove_label: Label not on card
Call `trello_remove_label` with `card_id` = `card_a_id`,
`label` = `label_3` (a label that was never added to Card A).
- **Verify:** Tool error returned. Error message indicates the label is
  not on the card.

---

## Phase 13: Cleanup

### 13.1 -- Archive all cards
Call `trello_archive_card` for each of the 4 card IDs.
- **Verify:** All return `archived` = true.

### 13.2 -- Inform the user
Tell the user:
- All cards have been archived.
- The lists and boards must be cleaned up manually in the Trello UI
  (list archiving and board deletion are not supported by trello-mcp).

---

## Tool coverage checklist

Before producing the final report, confirm every tool was exercised:

| Tool | Tested in |
|------|-----------|
| trello_boards | 0.1, 0.2 |
| trello_lists | 0.4, 1.5 |
| trello_cards | 0.5, 2.6, 2.7, 9.2, 9.3, 9.5, 10.1-10.4 |
| trello_get_card | 2.5, 3.1-3.5, 5.3, 5.5, 6.4, 8.1, 8.3-8.4 |
| trello_create_card | 2.1-2.4, 12.1 |
| trello_update_card | 3.1-3.5, 12.3 |
| trello_archive_card | 9.1, 13.1 |
| trello_unarchive_card | 9.4 |
| trello_add_comment | 6.1 |
| trello_search | 0.7, 7.1, 7.2 |
| trello_checklists | 4.3, 4.8 |
| trello_check_item | 4.6, 4.7 |
| trello_add_checklist | 4.1, 4.2 |
| trello_add_check_item | 4.4, 4.5 |
| trello_labels | 0.6 |
| trello_add_label | 5.1, 5.2, 12.2 |
| trello_remove_label | 5.4, 12.4 |
| trello_add_attachment | 6.2, 6.3 |
| trello_create_list | 1.1-1.4 |
| trello_board_summary | 0.3, 11.1 |
| trello_move_card | 8.1-8.4 |

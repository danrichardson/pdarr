# Chain Review — 2026-04-10

1 ticket implemented, ready for verification.

Preview: no preview configured — deploy to target machine and test manually.

## How to use this checklist

Walk through each ticket below. Check each verification step as you go.
Record your verdict at the bottom of each ticket section.
When done, run `/tsh TKT-010` to ship, or `/td TKT-010 {reason}` to defer.

---

## TKT-010: Review count and space saved not updating after files are reviewed

**Branch:** `ticket/tkt-010-review-count-space-saved`
**Files changed:** 3 (+ rebuilt frontend dist) | **Tests added:** 0 (no frontend test infra) | **Risk:** low

### What changed
Added a React context (`LayoutContext`) that shares the originals count between the Layout sidebar and child pages. The Review page now calls `refreshOriginals()` after every delete, restore, and bulk-delete action, which triggers the Layout to re-fetch the count and update the sidebar badge immediately.

### Acceptance Criteria
- [ ] After all review items are approved/dismissed, the sidebar review count badge resets to 0
- [ ] After reviewed files are accepted, the "Space Saved" dashboard metric updates to reflect the new totals
- [ ] Both the review count and space saved update without requiring a page refresh

### Verification Steps
- [ ] Navigate to Review page — confirm the sidebar badge shows the current count of pending originals
- [ ] Click the delete (trash) button on one file — confirm the sidebar badge decrements by 1 without a page refresh
- [ ] Click restore on another file — confirm the sidebar badge decrements by 1
- [ ] Use "Select all" then "Delete N originals" → confirm bulk delete — sidebar badge should drop to 0
- [ ] Navigate to Dashboard — confirm "Space Saved" shows the correct cumulative value
- [ ] Refresh the browser — confirm the review count badge is still 0 (or whatever the correct count is)

### Edge Cases
- [ ] If there are 0 originals pending, the Review badge should not appear in the sidebar at all (no "0" badge)
- [ ] If an API call fails (e.g., delete fails), the badge should stay at its previous value (not decrement prematurely)

### Regression Checks
- [ ] Queue page still works — jobs still appear and update
- [ ] Dashboard stats (CPU, encoder info) still render correctly
- [ ] Mobile bottom nav still shows the review badge correctly
- [ ] SSE-based progress updates on Queue/Dashboard still work

### Verdict
- [ ] pass
- [ ] fail — reason:

---

## Results

### TKT-010: Review count and space saved not updating after files are reviewed
- [ ] pass
- [ ] fail — reason:

**When ready:** `/tsh TKT-010` to ship.
**To defer:** `/td TKT-010 {reason}`
**To clean up:** `/tcl --all`

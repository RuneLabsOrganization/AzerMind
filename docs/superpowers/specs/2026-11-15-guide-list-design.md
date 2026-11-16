# Guide references list

Replace the multiline editor with up to twenty individually editable URL rows.
Each row has a number, domain or validation feedback, a readable single-line
input and a minus button. A visible Add guide action appends and focuses a row,
scrolling it into view. Tab/Shift-Tab move between rows; Enter moves to the next
row or adds one; Escape releases input focus. Empty state explains the plus.

Keep the existing newline-delimited saved format and SaveGuideReferences API.
Preserve full URLs, query strings, variants, order and existing saved references.
Blank rows are drafts and ignored when saving. Validate HTTP(S), duplicates,
individual and combined length and twenty-item limit without fetching any URL.
Save persists changes; Cancel reloads saved rows; switching categories and closing
continue to protect unsaved drafts. Removing a saved row only takes effect on
Save. No changes to character consent or historical messages.

Test add/remove/edit/save/cancel, empty and full lists, URLs with variants,
invalid and duplicate rows, draft preservation and keyboard focus/scrolling.
Use the existing shared settings frame and Grimoire components. Dismiss stale
tooltips when changing categories. Validate and install with backup.

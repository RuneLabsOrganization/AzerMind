# Spell and action bar context

The player wants AzerMind to recommend an arrangement using learned abilities,
current buttons and hotkeys. Existing learned-spell context repeats ranks and
can truncate before covering the whole spellbook; it has no button mappings.

## Design

- Add one read-only Lua adapter, `ActionBarContext.lua`, loaded after the core.
- Attach its snapshot only to relevant bar, keybind or ability-organization
  questions, before link expansion can introduce unrelated trigger words.
- Respect character sharing and the existing spells checkbox, renamed to make
  action-bar sharing explicit. Basic/off and spells-disabled send no snapshot.
- Compact active player spellbook entries by highest reported rank where known.
  Keep missing ranks unknown; filter passive and off-spec entries when exposed.
- Inspect visible default player buttons using their actual mapped slot. Include
  action identity/rank and effective keys. Distinguish empty from unavailable.
  Macros remain opaque; never read macro bodies or change buttons or bindings.
- Bound the attachment to 5,000 UTF-8 bytes with complete lines and explicit
  partial/coverage notices. Hidden pages and custom, pet, stance and vehicle bars
  are outside this first adapter. Missing keys are not evidence of free keys.
- Keep the existing 48,000-byte question and 63,000-byte multipart budget.
  An oversized attachment is omitted explicitly, preserving the question.

## Implementation and verification

1. Agent owns the new adapter and its Fengari tests: red first, then scoped reads,
   rank compaction, actual slot/paging, keybinds, missing APIs and byte bounds.
2. Root owns send-hook integration, TOC/test loaders, settings caption, docs and
   package test entry. Test attachment order, failure isolation and size limits.
3. Root performs one consolidated review, fixes findings directly, and runs the
   appropriate full suite. Publish this topic in its own PR.
4. Cherry-pick only this feature into the live checkout; back up and copy changed
   addon files. Verify installed hashes. A new Lua module requires fully restarting
   the client before live validation; do not close an active game without notice.

No background polling, bridge process change or extra addon dependency is needed.

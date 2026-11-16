# PR integration, 2026-11-15

Started from a clean `main` at `9e51e49`. Preserved that tip in
`backup/pre-integration-20261115`. The user explicitly limited this integration to
open PRs; the separate clean `feat/go-companion` worktree remains unchanged.

Integrated sequentially:

1. PR #9, `feat/action-bar-context`, head `fbbd8ad`: no textual conflicts.
   Reviewed message enrichment, display filtering, consent and load order.
   Lua checks and all 101 focused action-bar/addon tests passed.
   Merge commit: `364dcc5`.
2. PR #10, `fix/dungeon-alias-pronouns`, head `e76b4b4`: no textual conflicts.
   Preserves ordinary lowercase dungeon abbreviations while preventing common
   Portuguese words from becoming unintended dungeon destinations.
   All 21 journal/gear tests passed. Merge commit: `16776b2`.
3. PR #11, `feat/route-intelligence`, head `0dbc325`: conflicts with #9 in
   README.md, AzerMind.lua, package.json and tests/addon_test.js.
   Combined both documentation entries, sharing labels, raw-question adapters,
   attachment budgets, hidden message blocks, test cases and module loaders.
   Retained both action-bar and route observations; neither replaces the other.
   The shared Grimoire test VM also loads all newly integrated modules.

Final validation: all 308 Node tests passed, all Lua 5.1 parse/declaration-order
checks passed, and all 16 real Windows codec cases passed. Diff whitespace checks
passed. Existing Grimoire assets/theme, compact auction tooltips, credits settings
and item presentation are unchanged from `9e51e49`.

No live installation or in-game visual approval is claimed by these automated
checks. New Lua modules need an updated installation and a full WoW restart.

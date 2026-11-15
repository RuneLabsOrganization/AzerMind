# Route intelligence from actual progress

The player wants short next steps when asking where to go, grounded in an authored
route and current quest progress. Existing answers contain generic travel and
insufficient quest endpoint facts. The selected approach reads an activated
RestedXP guide and reconciles it against WoW; it does not implement a decrypter or
publish imported guide packs. Install the official RXP addon separately and let
its normal importer validate the supplied file for the account.

## Components and ownership

- `RestedXPContext.lua` and its focused tests: agent owns the optional read-only
  adapter. Read active parsed guide and matching progress fields, never call
  importer, progress mutators or arbitrary condition functions. Expose a bounded
  snapshot of current and a few following steps, with future steps conditional.
- `RouteQuestContext.lua` and tests: agent owns quest endpoint evidence. Read active
  objectives and readiness, check delivered flags afresh, enrich relevant ready
  quests with finisher names and exact mapped coordinates from local catalog.
  Quest category is not a turn-in location. Preserve unavailable/partial data.
- Root owns query routing, sending, UI text filtering, prompt policy, docs, package
  tests and installation. Respect character/quests sharing and location consent;
  attach only to relevant next-step/route questions, bounded in bytes.

## Behavior

Give a short ordered set of actionable next steps. Source the ordering from the
active applicable guide when available; otherwise ground a clearly labeled local
plan in known active/ready quests. Skip delivered objectives; distinguish quest
completion, turn-in readiness, RXP skipped steps and unverified future steps.
Never promise fastest routing, assume learned flight paths, use quest-category
coordinates for turn-ins, or treat absence from completed history as eligibility.
Retain preferred/avoided areas expressed by the player. Only draw maps when asked.

## Verification and delivery

Use failing regressions for the current bad examples, including Regthar #1361:
Desolace category, Barrens finisher. Check unavailable RXP, mismatched guide progress,
skipped steps, changed completed flags, malformed data, byte bounds and no mutation.
Root performs one consolidated review and fixes findings, then tests the combined
feature and installed checkout. The unrelated Portuguese `vc`/Deadmines alias bug
is a separate PR. Copy only changed addon files with backup and verify installation.
Live routing verification requires a valid imported and selected guide; report
that dependency explicitly rather than claiming a successful import from format.

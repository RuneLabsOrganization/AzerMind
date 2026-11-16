---
status: verifying
trigger: "WoW told the user to disable AzerMind after a reported /target attempt; addon now disabled"
created: 2026-11-15
updated: 2026-11-15
---

## Current Focus

hypothesis: The /r interceptor makes native secure-command dispatch run through addon-owned SendText.
test: Preserve identities of native chat methods at login; run addon regressions after removing interception.
expecting: Native SendText, SendMessage and ProcessChatType remain unchanged while /ai and reply links remain usable.
next_action: Install the corrected addon with backup and re-enable it after WoW closes; actual client confirmation remains necessary.

## Symptoms

expected: AzerMind remains usable without blocked protected actions.
actual: WoW offered to disable AzerMind and user reports losing addon access.
errors: Exact API and stack unknown; user recalls /target.
reproduction: Unknown.
started: Reported after PR integration; live installation version being investigated by parent.

## Eliminated

- hypothesis: Direct literal target execution in current AzerMind code.
  evidence: No /target, TargetUnit or RunMacroText call found in source or installed source (parent checked).

## Evidence

- checked: Source protected-action search.
  found: SetBinding in settings and slash binding command; macro creation/update/pickup in explicit install flow.
  implication: Need inspect guarding and auto-call paths; no root cause established.
- checked: Complete HookReplyCommand and login call path.
  found: On login, all chat editboxes have SendText, SendMessage and ProcessChatType replaced with addon closures. Non-agent messages call originals inside those closures.
  implication: Ordinary player commands including /target enter native secure dispatcher through addon Lua.
- checked: Blizzard Forever ChatFrameEditBox.lua (Gethe source mirror), lines 228-236 and 275-277.
  found: SendText invokes ParseText; ParseText directly invokes secure command handler before ordinary chat types.
  implication: Wrapping SendText contaminates the protected command dispatch path. A literal /target in AzerMind is unnecessary.
- checked: New regression with native chat methods populated before PLAYER_LOGIN.
  found: Regression failed before fix because ProcessChatType identity changed; passed after removing native method interception.
  implication: Unsafe source behavior reproduced and removed. Exact original live event remains unavailable because no taint log exists.

## Resolution

root_cause: Identified unsafe native chat SendText/SendMessage/ProcessChatType replacement, a direct taint path into protected player slash commands. Original live occurrence has no captured stack.
fix: Removed agent /r interception. /r remains native; explicit /ai and addon reply links remain available.
verification: Failing-before/passing-after native-method regression; all 309 Node tests, Lua parse/order checks and all 16 Windows codec cases passed. Actual WoW reproduction remains unverified.
files_changed: [addon/AzerMind/AzerMind.lua, tests/addon_test.js]

# Live character context synchronization

The bridge was stopped and retained a level-38 snapshot. Restarting it and receiving a new Connect updated the live state to level 39. The current addon additionally treats unrelated bridge presence as acknowledgement of a context-bearing hello, which can discard a fragmented snapshot before receipt. It has no background context synchronization.

## Design

Reuse the existing pixel and reply transports. A context-only `h;u;c` record cannot launch an AI request, including on older bridges that recognize `h`. The current bridge recognizes `u`, persists the context, and returns a `context-ack` with the exact addon session and message ID. Normal hellos receive `hello-ack`. Presence alone must never acknowledge a snapshot. Duplicate controls reissue receipts without overwriting newer context.

Every 30 seconds, when connected, outside combat, and with no outstanding transmission, the addon compares fresh consent-filtered context with the last acknowledged snapshot. Changed context is sent without chat history, AI calls, or gameplay input. Pending synchronization is bounded, coalesces naturally, and cannot trigger a UI reload or mark user chat as failed. Incoming acknowledgements cannot regress the last acknowledged snapshot.

Bridge state preserves `context.at` as content-change time and records receipt time separately. No game memory access, input automation, or new addon file is introduced. Loading the updated existing Lua file requires one user-initiated `/reload`.

## Implementation and verification

1. Establish baseline addon/protocol tests and reproduce the false acknowledgement before editing.
2. Add protocol/control integration tests for context-only snapshots, duplicate receipts, session isolation, clearing, and no AI execution; then implement the bridge changes.
3. Add addon tests for unrelated presence, exact receipts, deferred/coalesced updates, consent, combat, and chat priority; then implement addon scheduling.
4. Perform one consolidated review, resolve findings directly, and run the relevant full suite and Windows codec checks.
5. Back up and install only changed runtime files, restart one bridge process, and verify installed hashes. Ask the user to reload once; verify a fresh passive snapshot via logs/state without controlling the game.

The Windows bridge still needs a running helper. Automatic Windows startup is outside this patch; document the distinction between Connect and starting the helper.

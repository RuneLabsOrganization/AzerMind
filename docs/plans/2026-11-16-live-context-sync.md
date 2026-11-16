# On-demand character context delivery

Character context can be lost if generic bridge presence is treated as acknowledgement before a fragmented hello is received. The resulting old level/talent snapshot must not be mistaken for live data.

## Design

Keep the existing pixel and reply transport with exact message/session acknowledgements. Login and Connect carry fresh consent-filtered context. User questions build current context at send time and include it whenever it differs from the last acknowledged snapshot. Ordinary gameplay does not scan or send character context on a timer. Incoming acknowledgements cannot regress the last acknowledged snapshot.

Explicit sharing-setting changes cancel obsolete pending announcements. Their clear/filtered context stays pending until delivered, including on the next question or Connect after disconnection. No periodic retry is created for changed game state.

The bridge retains support for context-only controls from compatible addon versions, returns explicit receipts without launching AI jobs, and keeps content-change and receipt timestamps separately. An idle saved snapshot is historical until the next user-triggered refresh. No process-memory reads or game input are introduced.

## Verification and installation

- Drive the real Lua in the stub game VM: after acknowledged login, changing level, talents, quests, position and progress across repeated long timer advances must not read context or emit any pixel records.
- Sending a question must immediately include the current full consent-filtered state.
- Preserve receipt/session isolation, stale signal protection, pending consent clears, hello expiry and chat transmission regressions.
- Root owns production Lua and documentation; the bounded test agent owns addon tests. Perform one consolidated review for the final behavior and run the relevant suites.
- Back up and install only the changed existing Lua file, verify its hash and request one safe user reload. No bridge restart is required.

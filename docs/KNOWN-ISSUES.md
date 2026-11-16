# Known issues

## Character context stays at an old level or talent allocation

The bridge keeps its last snapshot across restarts. A saved snapshot is not proof that the helper is running or that the addon has sent current data. **Connect** sends a handshake from the addon; it cannot launch the Windows helper. Start the bridge with `npm start` or the Windows launcher, then keep WoW visible long enough for the pixel transmission to finish. Starting a second bridge against the same configuration can make the processes compete for the same files.

Before the context synchronization fix, character data was sent only at login, Connect, or with a question. An unrelated presence heartbeat could also acknowledge a fragmented hello prematurely, removing its context before the bridge received it.

The fix sends changed, consent-filtered context every 30 seconds while connected, outside combat, and with the transport idle. These records update the local snapshot without starting an AI request. Message-specific acknowledgements replace the presence shortcut. Retries and out-of-order delivery cannot overwrite a newer snapshot.

Install both the addon and bridge changes, restart the bridge, then use `/reload` when safe. No new addon folder is required. In `bridge/bridge.log`, look for `game context updated` and `context sync #...`. In `bridge/state.json`, `context.at` is the last content change, `context.receivedAt` is its last accepted receipt, and `contextReceipt` identifies the last accepted context-bearing message. Unchanged snapshots are not resent; these timestamps can therefore remain stable while the character is idle. An empty shared context clears `context`, while the receipt metadata remains for ordering.

This transport still requires the helper and a visible, unobstructed pixel strip. It does not read game memory or control gameplay. If the helper is stopped, the last saved snapshot remains historical until a new transmission arrives.

## Sending message stalls on Windows

On an affected Windows setup, the addon can stay on **Sending message** while the pixel strip is visible and the bridge reports `capture: attached to ...`. The console may show repeated `strip seen but rejected: checksum` warnings or no matching `strip #N` line. A green connection light means the addon has recently heard from the bridge through a reply slot or presence signal. A matching `(pixel) ... starting` line confirms that an outgoing request reached the bridge.

### Diagnose and recover

1. Keep WoW visible in windowed or borderless mode with the top-left corner unobstructed. Check the bridge console or its redirected stdout for `strip #N`, followed by the request's `(pixel) ... starting` line.
2. Run `/azermind diag` in game to check the sound channel, reply slot polls and connection state. When sharing diagnostics, redact session tokens, private paths and chat text.
3. If the request stays on **Sending message**, run `/reload` when out of combat. The addon saves its pending outbox to SavedVariables for the bridge to read. A matching `(reload) ... starting` line confirms that the fallback delivered the request.

If the reply says **Selected model is at capacity**, the bridge has already dispatched the request to the AI agent. The provider can return that error after either a pixel or a reload submission. Retry later or select another model supported by the installed CLI and account. If changing `agents.codex.model` in `bridge/config.json`, restart the bridge for it to take effect.

### Cause and fix

On the affected machine, Windows GDI returned RGB channel values of 0 for off pixels and 64 for on pixels. The fixed threshold of 128 read both as off. The decoder now calibrates each channel from the known header cells when the default threshold misses the header. It requires sufficient contrast and still checks the full header, length and checksum.

For large frames, decode time plus the capture sleep could exceed the 750 ms each fragment remains visible. Moving the Windows decoder to C# reduced the median decode time for a 2,392-byte frame from 672 ms to 4.2 ms on Windows PowerShell 5.1.

As of 2026-11-12, both Windows reader fixes are in an open pull request. The change passed 164 Node tests and 16 codec cases. In WoW, a live test produced `strip #29`, a `(pixel) ... starting` entry, and `#29 ... done`. The reply appeared in the addon. [Windows capture fix](https://github.com/mateusmaaia/AzerMind-wow/pull/5)

A separate open pull request refreshes the pixel strip scale after the game window is resized. [Window resize fix](https://github.com/mateusmaaia/AzerMind-wow/pull/6)

To apply the Windows reader fix, update to a build containing PR #5 and restart the bridge. For the resize fix, also install the addon files from a build containing PR #6 and run `/reload`. Until the PRs are merged, updating the default branch alone does not include these fixes. A successful `/reload` fallback does not verify the pixel transport: confirm a new request reaches `(pixel) ... starting` and its reply appears without another reload.

## Native chat commands blocked as an AzerMind action

An older AzerMind build replaced Blizzard chat editboxes' `SendText`,
`SendMessage` and `ProcessChatType` to redirect `/r` to the agent. Normal player
commands also passed through those addon functions, creating an unsafe path into
protected command dispatch. WoW may offer to disable the addon when it blocks
such an action. This does not require a literal `TargetUnit` call in AzerMind.

The fix removes those replacements. `/r` now stays the normal player-whisper
reply; use `/ai <text>`, the addon input or `[reply]` to answer the agent.
Update the addon and fully restart WoW to clear the old wrappers, then enable
AzerMind. Chats and settings do not need to be reset. Automated regression checks
that native method identities survive login; actual client confirmation is still
required because the original warning had no captured stack.

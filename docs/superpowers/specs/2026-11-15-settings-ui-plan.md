# Settings UI implementation plan

Approved direction: rebuild Config as one framed Grimoire window, retaining the
existing settings capabilities and saved data. Work sequentially and verify each
step. No new art, protected game actions or native chat interception.

1. Build one opaque framed shell with a wider persistent navigation column,
   a content area, one close button, and a mouse-blocking dimmer behind it.
2. Move all seven existing categories into that shell. Preserve public page
   entry points and sharing consent; prevent navigation while dirty or saving.
   Use one Escape/close path and retain drafts when closing the main chat.
3. Reorganize AI scope, inheritance, selectors and footer. Show inherited values
   legibly with an explicit Customize action; keep pending/read-only states
   distinct. Move model refresh alongside Model and technical revision to About.
4. Normalize all page headings, footer actions, scrolling and selector widths.
   Keep model search, custom IDs, reasoning capability checks, guide paste,
   character preview, sounds, bindings, opacity and copyable credits working.
5. Add regression coverage for borders, one-window navigation, disabled/pending
   states, draft protection, keyboard and geometry at supported sizes. Run the
   relevant suites, review the final diff, commit/push and install with backup.

Acceptance: no repeated close/back buttons; no navigation label wrapping;
visible active category; no controls overlapping footer/corners; inherited values
readable; existing settings persist with unchanged scope and consent. Automated
tests do not replace review of actual WoW screenshots.

Progress: plan recorded; implementation pending.

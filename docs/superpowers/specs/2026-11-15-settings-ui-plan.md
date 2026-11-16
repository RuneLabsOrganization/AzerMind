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

Progress: steps 1-5 implemented and checked sequentially. All seven public page
entry points use one shell; modal close/navigation guards preserve unsaved state
and pending saves. Inheritance stays readable with explicit customization. Page
headers, action footers, guide frame/scrollbar and appearance spacing are aligned.

Validation:
- The complete suite passed 316 Node tests and all 16 real Windows codec cases.
- After final state/feedback adjustments, all 161 UI checks passed, along with
  focused settings/auction tests and Lua 5.1 syntax/declaration-order checks.
- Screen-fit fixtures cover 900x600, 1280x720 and 1920x1080; opacity inheritance,
  one close/escape path, draft protection and save/cancel persistence are covered.
- Installation was snapshotted; all 170 installed files matched the project.
- Actual WoW visual review remains pending: both computer-use entry points failed
  to initialize with a kernel-assets path error. No screenshot approval is claimed.

The final changes after validation are documentation and the About revision text.
A /reload loads the new Settings.lua; player conversations and saved preferences
are not migrated or reset by this UI change.

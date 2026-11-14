# Spell and action bar advice

Ask AzerMind **"Ajude a reorganizar minhas skills e atalhos"** or **"Help arrange
my action bars and keybinds"**. With character sharing and **Spells, action bars
and hotkeys** enabled in Character context settings, the question includes a
fresh read-only snapshot of:

- Active learned player spells, compacted to the highest reported rank per name
  where ranks are available. The ordinary character context still lists ranks.
- Visible default player action buttons, their currently mapped slot and action.
- The reported hotkeys for those buttons, including modifier combinations.

The AI can use this to propose a layout appropriate to the abilities you have.
AzerMind does not move buttons, change hotkeys, or read macro bodies. Macro
actions are opaque; explain their purpose yourself if needed. Rank-one utility
spells already placed on a bar retain their actual rank in the snapshot.

The snapshot is attached only to relevant questions. It is capped at 5,000 UTF-8
bytes and marks partial lists. Near the message limit it is omitted with a notice.
Disabling character sharing or the spells checkbox prevents collecting it.

## Coverage and validation

The first adapter reads the main bar and default multibars through their visible
buttons' actual mapped slots, so it does not assume the main bar is on page one.
Hidden buttons/pages and custom, pet, stance and vehicle bars are outside this
snapshot. Empty, unavailable and unbound are distinct. An omitted spell, button
or key is not evidence that it is unlearned, empty or free. Item names require
the client's existing item cache; uncached items retain their ID.

After installing this new Lua module, fully exit and reopen WoW once. Then ask
the example question. Confirm the attached snapshot agrees with two visible
buttons and their keys; change a button or key yourself and ask again to confirm
the new snapshot. Automated tests check collection, bounds and transport; they
do not establish the live client's button/API compatibility.

Implementation: `addon/AzerMind/ActionBarContext.lua`, called from `AzerMind.Send`.
No new background process, polling timer or addon dependency is introduced.

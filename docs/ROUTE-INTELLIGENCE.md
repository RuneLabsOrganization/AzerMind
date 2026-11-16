# Next-step route intelligence

Ask **"Onde eu vou agora?"**, **"Onde upar?"**, **"Quais os próximos passos?"** or
**"What should I do next?"**. With character sharing and **Quests and route
progress** enabled, AzerMind attaches fresh route evidence to that question.

The answer should give a short sequence of actions with destinations and reasons.
An applicable selected RestedXP guide supplies the intended order; actual WoW
quest state decides whether a quest is already delivered, active or ready to
turn in. Future guide steps and chain successors retain their prerequisites.
Without an active guide, AzerMind can give a clearly labeled local suggestion
using known quests. It does not claim that this is an optimal leveling route.

## RestedXP setup

1. Install the official [RestedXP addon](https://github.com/RestedXP/RXPGuides/releases).
2. Fully restart WoW after adding new addon files.
3. Use RestedXP's normal **Import Guide** UI for your compatible guide file. Its
   importer validates the account and format. A generated code is not proof that
   importing or game-version checks will pass.
4. Select an appropriate guide and current starting point in RestedXP.
5. Ask AzerMind where to go next. Confirm the answer's guide and current step
   match the selected route.

AzerMind reads the active parsed guide and matching saved progress. It does not
decrypt imports, change licenses, select or advance guides, or call condition
functions that may change RestedXP settings. No guide pack is included in this
repository. RestedXP is optional; its code/license remain separate from AzerMind.

## Evidence and limits

- RestedXP: at most the current step and three subsequent conditional steps,
  with up to six supported elements per step and a 3,000-byte budget. RXP's
  runtime fields are internal and version-sensitive. Missing/mismatched progress
  is unavailable, never guessed. Source zone identifiers are not UI map IDs.
- Quest evidence: bounded active log reads, readiness, remaining objectives and
  recorded finishers from the bundled QuestieDB catalog, at most 3,500 bytes.
  Finisher geography is distinct from the quest category. Missing spawn data or
  incomplete prerequisites remain unknown. Authored endpoint coordinates are
  public catalog data; the adapter does not read player coordinates.
- Route attachments are only collected for relevant questions with quest sharing
  enabled. The combined attachment is under 7,200 UTF-8 bytes. Data is preserved
  in transport/history but hidden from the player message bubble. A near-full
  outgoing message receives an omission notice when space permits.
- RXP skip flags are not quest completion. Fresh positive WoW delivered flags
  take precedence. No learned flight network, exact travel time, XP/hour or
  optimal-path guarantee is inferred. Map markers still require a player request.

Development references: [RXP runtime/progress](https://github.com/RestedXP/RXPGuides/blob/cbe05e351e083108d08b35908347952fb0eaf06e/RXPGuides.lua),
[guide parser](https://github.com/RestedXP/RXPGuides/blob/cbe05e351e083108d08b35908347952fb0eaf06e/GuideLoader.lua).
Automated fixtures do not substitute for checking the selected live guide.

# Items browser redesign and iteration plan

Status: implementation in progress; previously completed UI work is preserved.
Starting point: clean `main` at `08ee4e0`.

## Goal

Make it easy to find an item, understand active filters and attach the intended
selection. Keep the existing Grimoire identity, owned/catalog sources, data
provenance and read-only game behavior. Implement and verify one stage at a time.

## Findings from the current screen and code

- Armor subclasses occupy the flat sidebar and also appear in Subtype.
- Category and result scrollbars crowd each other; the category list has short
  rows and nested native controls. Items lacks the framed shell used by Config.
- Search, source, location, rarity, slot, level, bulk selection and pagination
  compete at the top and bottom without clear grouping.
- The 760x600 layout reserves only a small results viewport. Item hit regions,
  checkbox, icon and text dimensions are specified separately.
- The summary calls all matched results "shown", despite paging and scrolling.
- Metadata can be unknown. Bank data is a snapshot when the bank is closed;
  database ownership filters refer to the player's possessions, not item source.
- Selection already survives filters and is separated by owned/catalog source.
  Preserve this. "Select page" is additive despite a misleading source comment.
- Existing item attachments have a 100-item/36,000-byte limit and target the
  conversation that opened the picker. Keep these protections and no auto-send.

## Design decisions

### One consistent window

Use a framed, opaque Grimoire window with one close control, title and a fixed
footer. Start with a roughly 920x640 logical layout; validate the actual content
bounds before fixing dimensions. Scale to fit 900x600, 1280x720 and 1920x1080
screens without overlapping controls. Reuse Theme components and existing assets.

Arrange the contents in this order:

1. My items / Item database, with a short explanation of the active source.
2. Search and a compact filter band; keep selection actions out of this area.
3. Expandable category tree on the left; breadcrumb, result count and item list
   on the right. Each area has its own reserved scrollbar gutter.
4. Result navigation immediately below the list.
5. Fixed selection summary, review action, Cancel and Add to message.

### Category semantics

Use stable category/subcategory keys based on classID/subclassID rather than
positions in a flat array. Armor expands to Cloth, Leather, Mail, Plate, Shields
and any other available subclasses. Weapons, consumables, materials and the
remaining families use the same structure. Prefer client names with explicit
fallback names for known IDs. Keep an Unknown/Other path for incomplete metadata.

Clicking a parent name selects the whole family and reveals its children. A
separate arrow expands/collapses without changing the current filter. A child
selects that subtype. Collapsing a parent preserves a selected child and the
breadcrumb identifies it. Expansion persists while browsing this window.

Accessories is a slot-based convenience group, not a second classification of
the database. Define it explicitly from the existing neck/ring/trinket/cloak
slots. Do not infer it from names. Where it overlaps the game's Armor class,
do not present category counts as disjoint subtotals that must add up.

Remove the separate Subtype selector once the tree covers its functionality.
Counts reflect the active source and other filters, excluding the selected
category itself, so expanding a parent remains useful. Compute counts with the
filtered result pass, not for every render/scroll. Unknown subtype data must not
make an item disappear from All items or Unknown/Other.

### Search, filters and feedback

- Search names and types by words, case-insensitively; recognize exact numeric
  item IDs and pasted item links. Preserve the full selected item payload.
- Search within the chosen category. Show a breadcrumb and a clear Search all
  items action when users need to broaden it; never silently reset filters.
- Keep location, rarity, slot and required Min/Max level as independent filters.
  In Item database, label location controls as owned-in filters.
- Show active filter values with an easy way to clear one or reset filters.
  Reset does not clear selected items. Min greater than Max gets an inline error.
- Keep unknown client metadata distinct from "no results". Display bank age and
  catalog availability as short contextual notices rather than technical headers.
- Provide clear empty states with useful recovery actions, not just a blank list.

### Results and selection

Use consistent rows with a checkbox, icon, readable name/quantity and a compact
second line for subtype, slot, required level and location when known. Keep the
current item tooltip, Shift-expanded auction details and right-click sources.
Long names stay within their row; the full value remains inspectable.

Distinguish total results, current page/range and selected count. Add a small
sort control for Name, Rarity, Required level and Location where useful; sort
unknown values last, with stable ties and without changing selection.

Select page and Select all results are explicitly additive. Keep the latter in
the selection area and prevent operations exceeding the 100-item attachment
limit, with an explanation. Clear selection affects the active source only.

Review selected shows all selected entries for this source, including those
outside current filters; returning to results restores the previous filters.
The footer reports hidden selections. Review is available before attachment,
without adding another mandatory confirmation. Source switches keep independent
selections. Refresh reconciles unavailable items and reports removals.

Reconcile by the existing stable keys: owned items retain location plus full item
payload, including suffix/modifiers; catalog entries retain catalog item ID.
The same base item in bags and bank is not one selection. Refresh quantities and
provenance from the current records rather than attaching stale row snapshots.
Numeric or pasted-link searches must not replace the selected item's payload.

## Iteration sequence and checks

### 1. Category tree and matching

- [x] Define stable family/subtype records and fallback labels.
- [x] Implement select-parent, select-child and independent expansion.
- [x] Replace the duplicate Subtype selector with a breadcrumb.
- [x] Check all represented item families, unknown metadata and accessory slots.

Automated checks: Armor -> Cloth excludes Leather; parent includes all its
children; equal subclass IDs under different classes never collide; expand/
collapse does not change selection or the selected filter; all items remain
reachable. Visual check: indentation, arrow targets, readable labels and scroll.

### 2. Window and layout

- [ ] Add the outer frame and group source/search/filter/result/selection areas.
- [ ] Widen the category column and results; reserve scrollbar gutters.
- [ ] Normalize row heights, pointer targets, headings and footer positions.
- [ ] Verify keyboard focus and Escape; leave native/protected game actions alone.

Automated checks: control bounds at supported screens, scrolling/clipping,
singular close, no footer overlap, no row allocation growth. Visual check: actual
WoW screenshots at normal and reduced sizes. Textures, font metrics and layering
cannot be certified by the Lua simulator alone.

### 3. Search and filters

- [ ] Add word search, exact IDs/item links and explicit Search all items.
- [ ] Group filters, active indicators, per-filter clearing and reset.
- [ ] Add useful no-results, invalid-range and unavailable-data states.
- [ ] Keep metadata refresh and bank snapshot wording accurate.

Automated checks: combined category/location/rarity/slot/level/search filters,
cache update invalidation, unknown values and reset preserving selection.
Visual check: find a known bag item, an armor subtype and a database item.

### 4. Results and ordering

- [ ] Align item rows and selected feedback; keep tooltips and source browsing.
- [ ] Separate result/page/selection counters and improve page navigation.
- [ ] Add stable sorting with unknown values last.
- [ ] Preserve bounded visible-row reuse for large catalogs.

Automated checks: at least 2,000 items, paging/scrolling endpoints, stable sort,
long names, quantities and no accidental selection when inspecting a source.
Visual check: names, quantities, badges and focus remain legible with real items.

### 5. Selection and attachments

- [ ] Group additive selection actions and enforce the 100-item limit clearly.
- [ ] Add Review selected without losing the browsing state.
- [ ] Make hidden/unavailable selections visible and keep sources independent.
- [ ] Preserve draft-only attachment, original chat binding and byte limit.

Automated checks: select across pages/categories; reset and source switch; review
hidden selections; inventory refresh/removal; limit boundaries; unchanged bank,
catalog and auction attachment provenance; no automatic message transmission.
Visual check: review mixed selections and confirm what Add to message includes.

### 6. Integrated review and delivery

- [ ] Exercise the full flow on My items and Item database.
- [ ] Run appropriate focused regressions after each stage; run the full suite,
  Lua checks and codec cases after integration, then review the final diff.
- [ ] Update usage docs and register new UI tests in normal/release checks.
- [ ] Commit verified stages; install with backup and verify installed files.
- [ ] Record the in-game result separately from automated test results.

## Checkpoint rule

For each stage, record what changed, tests/results, screenshot findings and any
remaining issue before starting the next stage. If a check fails, fix that stage
and rerun the relevant checks. Avoid layering another redesign on an unresolved
regression. A code/test pass may be recorded while actual WoW visual review is
pending; do not call that stage visually approved without real client evidence.

No SavedVariables reset, catalog replacement, new provider integration, native
chat interception or protected gameplay automation belongs to this work.

Plan review: approved. The review confirmed source-selection preservation,
category semantics and stage checks; stable identity/quantity reconciliation and
search-versus-payload separation were made explicit above.

Iteration 1: category-tree regression and four existing inventory/catalog regressions passed. Lua checks passed. Client visual review remains pending.

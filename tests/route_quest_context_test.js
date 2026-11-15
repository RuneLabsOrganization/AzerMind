'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const { lua, lauxlib, lualib, to_luastring, to_jsstring } = require('fengari');

const addon = path.join(__dirname, '../addon');
const read = file => fs.readFileSync(path.join(addon, file), 'utf8');
const records = (file, group, ids) => {
  const rows = read(file).split(/\r?\n/);
  return ids.map(id => {
    const row = rows.find(line => line.startsWith(`AzerMindCatalog.${group}[${id}]=`));
    assert.ok(row, `missing bundled record ${group} #${id}`);
    return row;
  }).join('\n');
};
const fixtures = [
  'AzerMind = {}',
  read('AzerMind_Data/Catalog.lua'),
  records('AzerMind_Data_QuestDetails/quests-2.lua', 'quests', [1136, 1361, 1362]),
  records('AzerMind_Data_World/npcs-3.lua', 'npcs', [3389, 3441]),
  read('AzerMind_Data_QuestDetails/Ready.lua'),
  read('AzerMind_Data_World/Ready.lua'),
  `
    quests = {
      {questID=1136, title="Frostmaw", isComplete=true},
      {questID=9999, title="Raptor Hunt", isComplete=false},
      {questID=1361, title="Regthar Deathgate", isComplete=true},
    }
    completed = {[1136]=true}
    objectives = {
      [9999] = {
        {text="Nest located", finished=true},
        {text="Raptor Eggs", numFulfilled=2, numRequired=5, finished=false},
      },
    }
    C_QuestLog = {
      GetNumQuestLogEntries = function() return #quests end,
      GetInfo = function(index) return quests[index] end,
      IsQuestFlaggedCompleted = function(id) return completed[id] == true end,
      GetQuestObjectives = function(id) return objectives[id] or {} end,
    }
    local areas = {[17]="The Barrens", [405]="Desolace", [1638]="Thunder Bluff"}
    C_Map = {
      GetAreaInfo = function(area) return areas[area] end,
      GetMapChildrenInfo = function() return {
        {name=areas[17], mapID=1413}, {name=areas[405], mapID=1443}, {name=areas[1638], mapID=1456},
      } end,
    }
    unsafeCalls = 0
    local function mutate()
      unsafeCalls = unsafeCalls + 1
      error("route context must not change game state or read player position")
    end
    CreateFrame, SelectQuestLogEntry, AddQuestWatch, AcceptQuest, CompleteQuest = mutate, mutate, mutate, mutate, mutate
    GetZoneText, GetSubZoneText, UnitPosition = mutate, mutate, mutate
    C_QuestLog.SetSelectedQuest, C_QuestLog.AddQuestWatch = mutate, mutate
    C_Map.GetBestMapForUnit, C_Map.GetPlayerMapPosition = mutate, mutate
    AzerMindMap = {MarkQuest=mutate}
    WorldMapFrame = {Show=mutate, SetMapID=mutate}
    function has(text, expected)
      assert(text:find(expected, 1, true), "expected " .. expected .. " in " .. text)
    end
    function missing(text, unexpected)
      assert(not text:find(unexpected, 1, true), "unexpected " .. unexpected .. " in " .. text)
    end
  `,
  read('AzerMind/Quests.lua'),
  read('AzerMind/Catalog.lua'),
].join('\n');

function check(code) {
  const L = lauxlib.luaL_newstate();
  lualib.luaL_openlibs(L);
  function run(source) {
    const status = lauxlib.luaL_dostring(L, to_luastring(source));
    if (status !== lua.LUA_OK) throw new Error(to_jsstring(lua.lua_tostring(L, -1)));
  }
  try {
    run(fixtures);
    const module = path.join(addon, 'AzerMind/RouteQuestContext.lua');
    if (fs.existsSync(module)) run(fs.readFileSync(module, 'utf8'));
    run('assert(type(AzerMind.RouteQuestContext) == "function", "Route quest context is missing")\n' + code);
    run('assert(unsafeCalls == 0, "route context changed game state or read player position")');
  } finally {
    lua.lua_close(L);
  }
}

test('reports the bundled finisher and exact Barrens position for ready Regthar Deathgate', () => {
  check(`
    local text = AzerMind.RouteQuestContext()
    has(text, "Ready to turn in #1361 Regthar Deathgate")
    has(text, "NPC #3389 Regthar Deathgate")
    has(text, "The Barrens")
    has(text, "45.34, 28.41")
    has(text, "UI map 1413")
    missing(text, "UI map 1443")
  `);
});

test('reads delivered flags afresh and leaves the current log unchanged', () => {
  check(`
    local text = AzerMind.RouteQuestContext()
    missing(text, "#1136")
    missing(text, "Frostmaw")
    completed[1361] = true
    text = AzerMind.RouteQuestContext()
    missing(text, "#1361")
    missing(text, "#1362")
    assert(quests[3].isComplete and #quests == 3 and objectives[9999][2].numFulfilled == 2)
  `);
});

test('falls back to one fresh completed-history read when per-quest flags are unavailable', () => {
  check(`
    local reads, ids = 0, {1136}
    C_QuestLog.IsQuestFlaggedCompleted = function() error("unavailable") end
    C_QuestLog.GetAllCompletedQuestIDs = function() reads = reads + 1; return ids end
    local text = AzerMind.RouteQuestContext()
    assert(reads == 1)
    has(text, "Ready to turn in #1361")
    missing(text, "#1136")
    ids[2] = 1361
    text = AzerMind.RouteQuestContext()
    assert(reads == 2)
    missing(text, "#1361")
  `);
});

test('distinguishes active objective progress from turn-in readiness', () => {
  check(`
    local text = AzerMind.RouteQuestContext()
    local ready = assert(text:find("Ready to turn in #1361", 1, true))
    local active = assert(text:find("Active #9999 Raptor Hunt", 1, true))
    assert(ready < active)
    has(text, "Raptor Eggs (2/5)")
    missing(text, "Nest located")
  `);
});

test('reports chain metadata conditionally and omits a delivered follow-up', () => {
  check(`
    local next = AzerMindCatalog.quests[1362]
    next.preQuestSingle = {1361}
    next.preQuestGroup = {1360, 1361}
    local text = AzerMind.RouteQuestContext()
    has(text, "Conditional chain follow-up #1362 The Kolkar of Desolace")
    has(text, "minimum level 30")
    has(text, "preQuestSingle: #1361")
    has(text, "preQuestGroup: #1360, #1361")
    has(text, "availability unverified")
    completed[1362] = true
    missing(AzerMind.RouteQuestContext(), "#1362")
  `);
});

test('retains readiness and objectives with missing catalog data', () => {
  check(`
    AzerMindCatalog = nil
    local text = AzerMind.RouteQuestContext()
    has(text, "Ready to turn in #1361")
    has(text, "finisher catalog unavailable")
    has(text, "Raptor Eggs (2/5)")
    has(text, "partial")
  `);
});

test('retains an exact catalog area and coordinates when the UI map is ambiguous', () => {
  check(`
    C_Map.GetMapChildrenInfo = function()
      return {{name="The Barrens",mapID=1413}, {name="The Barrens",mapID=9001}}
    end
    local text = AzerMind.RouteQuestContext()
    has(text, "The Barrens")
    has(text, "45.34, 28.41")
    has(text, "UI map unavailable")
    missing(text, "UI map 1413")
    missing(text, "UI map 9001")
  `);
});

test('keeps unknown quest readiness, objective data, and delivery state explicit', () => {
  check(`
    quests[2].isComplete = nil
    C_QuestLog.GetQuestObjectives = nil
    C_QuestLog.IsQuestFlaggedCompleted = nil
    local text = AzerMind.RouteQuestContext()
    has(text, "turn-in readiness unknown")
    has(text, "Current objectives unavailable")
    has(text, "delivered status unavailable")
    has(text, "partial")
    missing(text, "Ready to turn in #1361")
    local readLog = AzerMind.ActiveQuests
    AzerMind.ActiveQuests = function() return nil end
    has(AzerMind.RouteQuestContext(), "active quest log unavailable")
    AzerMind.ActiveQuests = readLog
  `);
});

test('bounds reads and output, retaining whole UTF-8 lines and a partial marker', () => {
  check(`
    quests = {}
    for i = 1, 60 do
      local id = 10000 + i
      quests[i] = {questID=id, title=string.rep("é漢", 59) .. "\\nextra", isComplete=true}
      local spawns = {}
      for pos = 1, 12 do spawns[pos] = {pos, 1} end
      AzerMindCatalog.quests[id] = {finishedBy={{id}}}
      AzerMindCatalog.npcs[id] = {name=string.rep("é漢", 59), spawns={[17]=spawns}}
    end
    local reads = 0
    local getFlags = C_QuestLog.IsQuestFlaggedCompleted
    C_QuestLog.IsQuestFlaggedCompleted = function(id) reads = reads + 1; return getFlags(id) end
    local text = AzerMind.RouteQuestContext()
    assert(reads <= 40 and #text <= 3500 and utf8.len(text))
    local _, ready = text:gsub("Ready to turn in", "")
    local _, coordinates = text:gsub("UI map 1413", "")
    assert(ready > 0 and ready <= 8 and coordinates > 0 and coordinates <= 3 * ready)
    has(text, "partial")
    missing(text, "\\nextra")
    missing(text, "12.00, 1.00")
  `);
});

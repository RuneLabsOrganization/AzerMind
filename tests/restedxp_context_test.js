'use strict';
const test = require('node:test');
const fs = require('node:fs');
const path = require('node:path');
const { lua, lauxlib, lualib, to_luastring, to_jsstring } = require('fengari');

const adapter = path.join(__dirname, '../addon/AzerMind/RestedXPContext.lua');
const fixture = `
  function Forbidden() forbiddenCalls = forbiddenCalls + 1; error("method called") end
  forbiddenCalls = 0
  RXP = { GetGuideProgress = Forbidden, IsStepShown = Forbidden,
    currentGuide = { key="route", name="Test route", group="Forever", version=3, steps={
      {stepId=101, elements={{tag="accept", questId=11}}},
      {stepId=102, active=true, elements={{tag="accept", questId=42},
        {tag="goto", zone=1413, x=43.6, y=59.9}, {tag="complete", questId=43, obj=2, text="Find the key"}}},
      {stepId=103, elements={{tag="turnin", questId=43, text="Talk to the guard"}}},
      {stepId=104, elements={{tag="accept", questId=51}}},
      {stepId=105, elements={{tag="goto", mapID=1413, x=50, y=20}}},
      {stepId=106, elements={{tag="turnin", questId=61}}},
    }} }
  RXPCData = { guideProgress={route={step=2, stepId=102}}, stepSkip={ [4]=true } }
  delivered = { [42]=true }
  C_QuestLog = { IsQuestFlaggedCompleted=function(id) return delivered[id] end }
  function Context(options) return AzerMind.RestedXPRouteContext(options) end
  function Contains(value, fragment) assert(value:find(fragment, 1, true), value) end
  function Omits(value, fragment) assert(not value:find(fragment, 1, true), value) end
`;

function check(code) {
  const L = lauxlib.luaL_newstate();
  lualib.luaL_openlibs(L);
  try {
    function run(source) {
      const status = lauxlib.luaL_dostring(L, to_luastring(source));
      if (status !== lua.LUA_OK) throw new Error(to_jsstring(lua.lua_tostring(L, -1)));
    }
    run('AzerMind = {}');
    if (fs.existsSync(adapter)) run(fs.readFileSync(adapter, 'utf8'));
    run('assert(type(AzerMind.RestedXPRouteContext) == "function", "route adapter missing")');
    run(fixture + '\n' + code);
  } finally {
    lua.lua_close(L);
  }
}

test('reports missing runtime and refuses mismatched saved progress', () => check(`
  local rxp = RXP
  RXP = nil
  Contains(Context(), "unavailable")
  RXP = rxp
  RXPCData.guideProgress = {different = {step=2}}
  RXPCData.currentGuideName, RXPCData.currentGuideGroup, RXPCData.currentStep = "Other", "Forever", 2
  Contains(Context(), "unavailable")
  Omits(Context(), "Find the key")
  RXPCData.guideProgress.route = {step=2, stepId=103}
  Contains(Context(), "unavailable")
  RXPCData.guideProgress = {}
  RXPCData.currentGuideName, RXPCData.currentStepId = "Test route", 102
  Contains(Context(), "current step 2")
  RXPCData.guideProgress.playerGUID = "OtherPlayer"
  RXP.player={guid="ThisPlayer"}
  Contains(Context(), "unavailable")
`));

test('uses bounded active and conditional route evidence with fresh quest completion', () => check(`
  local result = Context()
  Contains(result, "Test route")
  Contains(result, "current step 2")
  Contains(result, "RXP zone 1413")
  Contains(result, "mapID 1413")
  Contains(result, "Find the key")
  Contains(result, "objective 2")
  Contains(result, "conditional")
  Contains(result, "skipped by RestedXP")
  Omits(result, "quest #42")
  Omits(result, "quest #51")
  Omits(result, "quest #61")
  Omits(Context({position=false}), "43.6")
  delivered[42] = 1
  Contains(Context(), "quest #42")
  delivered[43] = true
  Omits(Context(), "quest #43")
  Contains(Context(), "43.6")
  assert(forbiddenCalls == 0 and RXP.currentGuide.steps[2].elements[1].questId == 42)
  assert(RXPCData.guideProgress.route.step == 2 and RXPCData.stepSkip[4] == true)
`));

test('bounds UTF-8 and treats malformed or overlong data as partial', () => check(`
  local step = RXP.currentGuide.steps[2]
  step.elements = {{tag="accept", questId=false, text=string.rep("é", 600)},
    {tag="unknownTag", text="text outside allowlist"},
    {tag="complete", questId=49, skip=true},
    {tag="complete", questId=50, completed=true}}
  for i=5,60 do step.elements[i] = {tag="complete", questId=i+100, text=string.rep("🙂", 99)} end
  local result = Context()
  assert(#result <= 3000 and utf8.len(result), result)
  Contains(result, "partial")
  Contains(result, "skipped by RestedXP")
  Contains(result, "marked complete by RestedXP")
  Omits(result, "text outside allowlist")
  Omits(result, "quest #159")
`));

test('drops complete lines at the byte limit across the whole route window', () => check(`
  local guide = RXP.currentGuide
  guide.name, guide.group = string.rep("🙂", 30), string.rep("é", 60)
  for step = 2, 5 do
    guide.steps[step].elements = {}
    for i = 1, 6 do
      guide.steps[step].elements[i] = {tag="complete", questId=step*100+i, obj=1, text=string.rep("🙂", 30)}
    end
  end
  RXPCData.stepSkip = {}
  C_QuestLog.IsQuestFlaggedCompleted = function() return false end
  local result = Context()
  assert(#result > 2750 and #result <= 3000 and utf8.len(result), result)
  for line in result:gmatch("[^\\n]+") do
    if line:find('guide text "', 1, true) then assert(line:sub(-1) == '"', line) end
  end
  Contains(result, "route partial")
`));

'use strict';

const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const { lua, lauxlib, lualib, to_luastring, to_jsstring } = require('fengari');

function context(setup = '') {
  const L = lauxlib.luaL_newstate();
  lualib.luaL_openlibs(L);
  function run(code) {
    const status = lauxlib.luaL_dostring(L, to_luastring(code));
    if (status !== lua.LUA_OK) throw Error(to_jsstring(lua.lua_tostring(L, -1)));
  }
  function ask(input) {
    run('RESULT = AzerMind.ActionBarQuestionContext(' + JSON.stringify(input) + ')');
    lua.lua_getglobal(L, to_luastring('RESULT'));
    const result = to_jsstring(lua.lua_tostring(L, -1));
    lua.lua_pop(L, 1);
    return result;
  }
  run('share, spells = true, true; AzerMind = { ContextSettings = function() return {spells = spells}, share end }');
  const source = path.join(__dirname, '../addon/AzerMind/ActionBarContext.lua');
  assert.ok(fs.existsSync(source), 'ActionBarContext.lua implements the question snapshot');
  run(fs.readFileSync(source, 'utf8'));
  run(setup);
  return { run, ask };
}

test('only relevant questions with spell sharing enabled read live abilities and buttons', () => {
  const v = context('reads = 0; GetNumSpellTabs = function() reads = reads + 1; return 0 end');
  for (const input of ['Where does this drop?', 'embarrassed about barkskin', 'What does stealth do?', 'organizar profissões']) {
    assert.equal(v.ask(input), '');
  }
  v.run('share = false');
  assert.equal(v.ask('Reorganizar minhas habilidades?'), '');
  v.run('share = true; spells = false');
  assert.equal(v.ask('What should my keybinds be?'), '');
  v.run('assert(reads == 0); spells = true; GetNumSpellTabs = nil');
  for (const input of ['Reorganizar minhas habilidades?', 'Quero organizar minhas skills', 'Pode ver as teclas?', 'What are my action bars?', 'Sugestão de atalho', 'Can you rearrange my spells?']) {
    const result = v.ask(input);
    assert.match(result, /action bars and keybinds/i);
    assert.match(result, /unavailable/i);
  }
});

test('reads the current mapped buttons, two keys and click fallback while preserving empty and unknown slots', () => {
  const v = context(`
    local function visible() return true end
    ActionButton1 = { action = 73, bindingAction = "CURRENT1", IsVisible = visible }
    ActionButton2 = { action = 74, IsVisible = visible }
    ActionButton3 = { action = 75, IsVisible = visible }
    ActionButton4 = { action = 76, IsVisible = visible }
    ActionButton5 = { action = 77, IsVisible = visible }
    ActionButton6 = { action = 78, IsVisible = function() return false end }
    MultiBarRightButton1 = { action = 25, IsVisible = visible }
    ActionButton7 = { action = 79, IsVisible = function() error("visibility unavailable") end }
    ActionButton8 = { action = 80, IsVisible = visible }
    local secret = {}
    issecretvalue = function(value) return value == secret end
    GetActionInfo = function(slot)
      if slot == 73 then return "spell", 10 end
      if slot == 74 then return "macro", 9 end
      if slot == 76 then return "item", 84 end
      if slot == 77 then return secret, secret end
      if slot == 78 then error("hidden button read") end
      if slot == 80 then error("action unavailable") end
      if slot == 25 then return "spell", 15 end
    end
    C_Spell = {
      GetSpellName = function(id) return id == 10 and "Frostbolt" or "Blink" end,
      GetSpellSubtext = function(id) return id == 10 and "Rank 4" or "" end,
    }
    C_Item = { GetItemInfo = function() return "Mana Gem" end }
    GetBindingKey = function(command)
      if command == "CURRENT1" then return "SHIFT-1", "CTRL-4" end
      if command == "ACTIONBUTTON3" then return "3" end
      if command == "CLICK ActionButton2:LeftButton" then return "BUTTON3" end
      if command == "MULTIACTIONBAR3BUTTON1" then return "ALT-F" end
    end
    C_KeyBindings = { GetBindingByKey = function(key, ...)
      assert(select("#", ...) == 0)
      return ({ ["SHIFT-1"] = "CURRENT1", ["CTRL-4"] = "CURRENT1", ["3"] = "OVERRIDE_BUTTON", ["BUTTON3"] = "CLICK ActionButton2:LeftButton", ["ALT-F"] = "MULTIACTIONBAR3BUTTON1" })[key]
    end }
    GetMacroInfo = function() error("macro body read") end
    GetMacroBody = GetMacroInfo
    PlaceAction = function() error("slot modified") end
    SetBinding = function() error("binding modified") end
  `);
  const result = v.ask('Pode sugerir onde colocar as habilidades na barra?');
  assert.match(result, /Main 1 \[slot 73\].*Frostbolt.*Rank 4.*#10.*SHIFT-1.*CTRL-4/);
  assert.match(result, /Main 2 \[slot 74\].*macro #9.*BUTTON3/);
  assert.match(result, /Main 3 \[slot 75\].*empty.*bindings unavailable/);
  assert.match(result, /Main 4 \[slot 76\].*Mana Gem.*#84.*unbound/);
  assert.match(result, /Main 5 \[slot 77\].*unavailable/);
  assert.match(result, /Main 8 \[slot 80\].*action unavailable/);
  assert.match(result, /Right 1 \[slot 25\].*Blink.*ALT-F/);
  assert.doesNotMatch(result, /slot 78|slot 79|hidden button read|body read|modified/);
  v.run('ActionButton1.action = 75');
  assert.match(v.ask('keybinds?'), /Main 1 \[slot 75\].*empty/);
});

test('selects the highest proven ranks, preserves unknown ranks, and bounds long UTF-8 inventories', () => {
  const v = context(`
    local items = {
      { name = "Frostbolt", spellID = 1, itemType = 1 },
      { name = "Frostbolt", spellID = 2, itemType = 1 },
      { name = "Frostbolt", spellID = 3, itemType = 1 },
      { name = "Passive", spellID = 4, itemType = 1, isPassive = true },
      { name = "Other spec", spellID = 5, itemType = 1, isOffSpec = true },
      { name = "Flyout", spellID = 6, itemType = 3 },
    }
    for index = 7, 120 do
      items[index] = { name = "Zé habilidade " .. index, spellID = index, itemType = 1 }
    end
    C_SpellBook = {
      GetNumSpellBookSkillLines = function() return 1 end,
      GetSpellBookSkillLineInfo = function() return {itemIndexOffset = 0, numSpellBookItems = #items} end,
      GetSpellBookItemInfo = function(index) return items[index] end,
      GetSpellBookItemName = function(index)
        local ranks = { "Rank 2", "Rank 7", "" }
        return items[index].name, ranks[index]
      end,
    }
    Enum = { SpellBookItemType = { Spell = 1 }, SpellBookSpellBank = { Player = 0 } }
  `);
  const result = v.ask('What are my hotkeys?');
  assert.match(result, /Frostbolt \(Rank 7\) #2/);
  assert.match(result, /Frostbolt \(rank unavailable\) #3/);
  assert.doesNotMatch(result, /Frostbolt \(Rank 2\)|Passive|Other spec|Flyout/);
  assert.match(result, /partial/i);
  assert.ok(Buffer.byteLength(result, 'utf8') <= 5000);
  assert.doesNotMatch(result, /�/);
});

test('the complete snapshot stays within the budget when both bars and spellbook are full',()=>{
  const v=context(`
    for _,prefix in ipairs({"ActionButton","MultiBarBottomLeftButton","MultiBarBottomRightButton","MultiBarRightButton","MultiBarLeftButton","MultiBar5Button","MultiBar6Button","MultiBar7Button"}) do
      for i=1,12 do _G[prefix..i]={action=i,IsVisible=function() return true end} end
    end
    GetActionInfo=function(i) return "spell",i end
    GetBindingKey=function() return "CTRL-SHIFT-F12","ALT-SHIFT-F11" end
    C_Spell={GetSpellName=function(i) return "Habilidade ação "..i end,GetSpellSubtext=function() return "Rank 9" end}
    GetNumSpellTabs=function() return 1 end
    GetSpellTabInfo=function() return "General",0,0,200 end
    GetSpellBookItemInfo=function(i) return "SPELL",i end
    GetSpellBookItemName=function(i) return "Outra habilidade ação "..i,"Rank 3" end
    IsPassiveSpell=function() return false end
  `);
  const result=v.ask('minhas barras');
  assert.match(result,/Partial action bars/);
  assert.match(result,/Partial active spellbook/);
  assert.ok(Buffer.byteLength(result)<=5000,Buffer.byteLength(result));
  assert.doesNotMatch(result,/�/);
});

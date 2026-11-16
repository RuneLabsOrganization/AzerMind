'use strict';
const test=require('node:test'),assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path');
const boot=require('./helpers/grimoire-vm');
test('quest window reserves navigation, filters, rows and attachment footer at supported screens',t=>{
 const vm=boot(t);vm.run(fs.readFileSync(path.join(__dirname,'helpers/grimoire-geometry.lua'),'utf8'));
 vm.run(`
  AzerMind.ActiveQuests=function() local q={};for i=1,40 do q[i]={id=i,title='A long quest title '..i,level=20,zone='A long quest area',objectives={}} end;return q end
  AzerMind.ShowQuestPicker();P=AzerMindQuestPicker
  assert(P.grimoireFrame.name=='frame-window' and P.grimoireMaterial.grimoireAsset=='leather')
  local row=P.rows[1]
  assert(not row.grimoireFrame,'quest rows must remain neutral')
  assert(GR_Bounds(row.title).right<GR_Bounds(row.map).left,'title overlaps map action')
  assert(GR_Bounds(row.map).right<GR_Bounds(row.actions).left,'row actions overlap')
  assert(GR_Bounds(AzerMindQuestPickerScroll).bottom>GR_Bounds(P.previous).top,'list overlaps pagination')
  assert(GR_Bounds(P.classSelector).right<GR_Bounds(P.search).left,'sidebar overlaps search')
  assert(GR_Bounds(P.levelToggle).right<GR_Bounds(P).right-150,'level toggle overlaps reset')
  assert(#P.rows<=7)
  for _,size in ipairs({{900,600},{1280,720},{1920,1080}}) do
   UIParent:SetSize(size[1],size[2]);for _,fn in ipairs(UIParent.hooks.OnSizeChanged or {}) do fn(UIParent) end
   assert(P:GetWidth()*P:GetScale()<=size[1]-32 and P:GetHeight()*P:GetScale()<=size[2]-32)
  end
 `);
});
test('quest layout keeps selection, page navigation and draft-only attachments intact',t=>{
 const vm=boot(t);
 vm.run(`
  AzerMind.ActiveQuests=function() local q={};for i=1,40 do q[i]={id=i,title='Quest '..i,level=20,zone='Forest',objectives={}} end;return q end
  AzerMind.ShowQuestPicker();P=AzerMindQuestPicker
  P.rows[1]:SetChecked(true);P.rows[1].scripts.OnClick(P.rows[1]);P.next.scripts.OnClick()
 `);
 assert.match(vm.value('P.pageLabel:GetText()'),/26-40 \/ 40/);
 assert.equal(vm.value('P.selected[1]'),'true');
 vm.run(`P.previous.scripts.OnClick();ATTACHED=nil;AzerMind.AppendQuestDraft=function(text,id) ATTACHED=text;CHAT=id;return true end;P.insert.scripts.OnClick()`);
 assert.match(vm.value('ATTACHED'),/Active quest:/);assert.match(vm.value('ATTACHED'),/Quest 1/);
 assert.equal(vm.value('CHAT'),vm.value('P.chatId'));
});

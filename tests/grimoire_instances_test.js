'use strict';
const test=require('node:test'),assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path');
const boot=require('./helpers/grimoire-vm');
const geometry=fs.readFileSync(path.join(__dirname,'helpers/grimoire-geometry.lua'),'utf8');
function fixture(vm){vm.run(`
 AzerMindInstanceData={}
 for i=1,20 do
  AzerMindInstanceData[i]={id=i,name=string.format('Instance %02d',i),type=i<11 and 'Dungeon' or 'Raid',location=i<11 and 'Forest' or 'Mountain',level='20-30',minLevel=20,maxLevel=30,source='Test snapshot',bosses={{name='Boss '..i,loot={{id=100+i,name='Special loot '..i}}}}}
 end
 AzerMind.ShowInstancePicker();P=AzerMindInstancePicker
 function Click(label) for _,b in ipairs(P.children) do if b.text==label and b.scripts.OnClick then b.scripts.OnClick(b);return end end;error('Missing control '..label) end
`);}
test('instance search combines words and finds recorded bosses and loot without item cache',t=>{
 const vm=boot(t);fixture(vm);
 assert.equal(vm.num('#AzerMind.FilterInstances("Special 115", "Raid", "Mountain")'),1);
 assert.equal(vm.num('AzerMind.FilterInstances("Special 115", "Raid", "Mountain")[1].id'),15);
 assert.equal(vm.num('#AzerMind.FilterInstances("Forest Boss", "Dungeon", "All", 60)'),0);
 vm.run('P.search:SetText("Special 115");P.Refresh()');
 assert.match(vm.value('P.rows[1].meta:GetText()'),/Boss \/ loot match/);
});
test('instance review ignores filters, restores page and preserves compact original-chat attachment',t=>{
 const vm=boot(t);fixture(vm);
 vm.run(`P.page=2;P.selected[1]=true;P.selected[15]=true;P.kind='Dungeon';P.Refresh();P.selectedToggle:SetChecked(true);P.selectedToggle.scripts.OnClick(P.selectedToggle)`);
 assert.equal(vm.num('#P.visible'),2);
 assert.match(vm.value('P.selectionCount:GetText()'),/1 hidden/);
 vm.run('P.selectedToggle:SetChecked(false);P.selectedToggle.scripts.OnClick(P.selectedToggle)');
 assert.equal(vm.num('P.page'),2);
 vm.run(`ATTACHED=nil;AzerMind.AppendQuestDraft=function(text,id) ATTACHED=text;CHAT=id;return true end;P.attach.scripts.OnClick()`);
 assert.match(vm.value('ATTACHED'),/Instance 01/);assert.match(vm.value('ATTACHED'),/Instance 15/);
 assert(!vm.value('ATTACHED').includes('Boss:'));
 assert.equal(vm.value('CHAT'),vm.value('P.chatId'));
 vm.run('ATTACHED=nil;P.chatId=-1;P.attach.scripts.OnClick()');assert.equal(vm.value('ATTACHED'),'nil');
});
test('instance sorting and individual filter clearing preserve selection',t=>{
 const vm=boot(t);fixture(vm);
 vm.run(`P.selected[15]=true;P.kind='Raid';P.zone='Mountain';P.Refresh();P.filters.scripts.OnClick();local r=AzerMindChoiceMenu.rows[1];r.scripts.OnClick(r)`);
 assert.equal(vm.value('P.kind'),'All');assert.equal(vm.value('P.zone'),'Mountain');
 assert.equal(vm.value('P.selected[15]'),'true');
 vm.run(`AzerMindInstanceData[15].minLevel=1;P.sort='level';P.Refresh()`);
 assert.equal(vm.num('P.visible[1].id'),15);
 vm.run(`Click('Reset filters')`);assert.equal(vm.value('P.selected[15]'),'true');assert.equal(vm.value('P.zone'),'All');
});
test('instance layout reserves result, detail-action and footer areas and uses bounded rows',t=>{
 const vm=boot(t);fixture(vm);vm.run(geometry);
 vm.run(`
  assert(P.grimoireFrame.name=='frame-window')
  assert(#P.rows==6)
  for _,row in ipairs(P.rows) do
   local name,button=GR_Bounds(row.name),GR_Bounds(row.details)
   assert(name.right<=button.left,'instance name overlaps Details')
  end
  assert(GR_Bounds(P.rows[6]).bottom>GR_Bounds(P.pageLabel).top,'rows overlap paging')
  assert(GR_Bounds(P.attach).right<=GR_Bounds(P).right-30,'attachment outside frame')
  for i=1,10 do P.page=i;P.Refresh() end;assert(#P.rows==6)
  AzerMind.ShowInstanceDetails(AzerMindInstanceData[1],P);D=AzerMindInstanceDetails
  assert(D.grimoireFrame.name=='frame-window')
  assert(#D.bossRows==7 and #D.lootRows==5)
  assert(GR_Bounds(D.bossTitle).right<=GR_Bounds(D.npc).left,'boss title overlaps NPC action')
 `);
});

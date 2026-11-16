'use strict';
const test=require('node:test'),assert=require('node:assert/strict');
const fs=require('node:fs'),path=require('node:path');
const boot=require('./helpers/grimoire-vm');

test('all public categories share one framed window, one close control and persistent navigation',t=>{
 const vm=boot(t);
 vm.run(`
  for _,entry in ipairs({{"ai",AzerMind.CodexPrompt},{"guides",AzerMind.ShowGuideSettings},
    {"character",AzerMind.ShowContextSettings},{"notifications",AzerMind.ShowNotificationSettings},
    {"keys",AzerMind.ShowKeySettings},{"appearance",AzerMind.ShowAppearanceSettings},{"credits",AzerMind.ShowCreditsSettings}}) do
   entry[2]()
   local shell=AzerMindSettingsFrame; local page=shell.activePage
   assert(shell:IsShown() and page:IsShown(),'category did not open')
   assert(page:GetParent()==shell and page.settingsKey==entry[1],'category opened a separate window')
   assert(shell.grimoireFrame.name=='frame-window','outer ornament missing')
   assert(shell.navigation.grimoireFrame.name=='frame-panel','navigation border missing')
   assert(shell.navButtons[entry[1]].selectedState,'selected category unclear')
   for _,button in pairs(shell.navButtons) do assert(button:GetWidth()>=164 and button.caption.wordWrap==false,'navigation label may wrap') end
   for _,child in ipairs(page.children) do
    if child.text=='X' or child.text=='Close' or child.text=='Back' then assert(not child:IsShown(),'duplicate close/back control') end
   end
  end
  local count=0;for _,name in ipairs(UISpecialFrames) do if name=='AzerMindSettingsFrame' then count=count+1 end end
  assert(count==1,'duplicate Escape registration')
 `);
});

test('inherited AI values stay readable and customization unlocks actual selectors',t=>{
 const vm=boot(t);
 vm.run('AzerMind.CodexPrompt()');
 assert.equal(vm.value('AzerMindGeneralSettings.customize:IsShown()'),'true');
 assert.equal(vm.value('AzerMindGeneralSettings.model:IsEnabled()'),'false');
 assert.equal(vm.num('AzerMindGeneralSettings.model:GetAlpha()'),1);
 assert.ok(vm.num('AzerMindGeneralSettings.model.caption.textColor[1]')>0.8);
 vm.run('AzerMindGeneralSettings.customize.scripts.OnClick()');
 assert.equal(vm.value('AzerMindGeneralSettings.inherit'),'false');
 assert.equal(vm.value('AzerMindGeneralSettings.model:IsEnabled()'),'true');
 assert.equal(vm.value('AzerMindGeneralSettings.dirty'),'true');
 vm.run('AzerMindSettingsFrame.navButtons.credits.scripts.OnClick()');
 assert.equal(vm.value('AzerMindSettingsFrame.activePage.settingsKey'),'ai');
 assert.match(vm.value('AzerMindGeneralSettings.status:GetText()'),/Save or cancel/);
});

test('close, Escape and dimmer protect drafts while a pending save cannot close the shell',t=>{
 const vm=boot(t);
 vm.run(`
  AzerMind.ShowGuideSettings();local p=AzerMindGuideSettings
  p.input:SetText('https://example.com/unsaved');p.input.scripts.OnTextChanged()
  AzerMindSettingsFrame.dimmer.scripts.OnMouseDown()
 `);
 assert.equal(vm.value('AzerMindSettingsFrame:IsShown()'),'true');
 assert.equal(vm.value('STUB.popup.which'),'AZERMIND_DISCARD_UI_CHANGES');
 assert.equal(vm.value('AzerMindGuideSettings.input:GetText()'),'https://example.com/unsaved');
 vm.run('AzerMindSettingsFrame:Hide()');
 assert.equal(vm.value('AzerMindSettingsFrame:IsShown()'),'true');
 vm.run('StaticPopupDialogs.AZERMIND_DISCARD_UI_CHANGES.OnAccept(nil,STUB.popup.data)');
 assert.equal(vm.value('AzerMindSettingsFrame:IsShown()'),'false');
 assert.equal(vm.value('AzerMindSettingsFrame.dimmer:IsShown()'),'false');
 vm.run('AzerMind.CodexPrompt(); AzerMindGeneralSettings.pending=true; AzerMindSettingsFrame:Hide()');
 assert.equal(vm.value('AzerMindSettingsFrame:IsShown()'),'true');
 vm.run('AzerMindSettingsFrame.close.scripts.OnClick()');
 assert.equal(vm.value('AzerMindSettingsFrame:IsShown()'),'true');
 assert.match(vm.value('AzerMindGeneralSettings.status:GetText()'),/Saving/);
});

test('AI fields and footer have non-overlapping bounds and the shell fits supported screens',t=>{
 const vm=boot(t);
 vm.run(fs.readFileSync(path.join(__dirname,'helpers/grimoire-geometry.lua'),'utf8'));
 vm.run(`
  AzerMind.CodexPrompt(); local p=AzerMindGeneralSettings
  for _,size in ipairs({{900,600},{1280,720},{1920,1080}}) do
   UIParent:SetSize(size[1],size[2])
   for _,fn in ipairs(UIParent.hooks.OnSizeChanged or {}) do fn(UIParent,size[1],size[2]) end
   local shell=AzerMindSettingsFrame
   assert(shell:GetWidth()*shell:GetScale()<=size[1]-32,'settings clipped horizontally')
   assert(shell:GetHeight()*shell:GetScale()<=size[2]-32,'settings clipped vertically')
  end
  local agent,model,reason=GR_Bounds(p.agent),GR_Bounds(p.model),GR_Bounds(p.reasoning)
  assert(agent.bottom>model.top and model.bottom>reason.top,'selectors overlap')
  local status,save=GR_Bounds(p.status),GR_Bounds(p.save)
  assert(status.bottom>save.top,'status overlaps fixed actions')
  assert(GR_Bounds(p.refreshModels).bottom>model.top,'model refresh overlaps model field')
  assert(GR_Bounds(p.agentLabel).bottom>agent.top,'label touches selector')
  assert(p.model.caption.wordWrap==false,'long model names may break control layout')
 `);
});

test('character Save and Cancel preserve category, consent and persisted values',t=>{
 const vm=boot(t);
 vm.run(`
  AzerMind.ShowContextSettings(); local p=AzerMindContextSettings
  ORIGINAL=p.enabled
  p.enabled=not ORIGINAL;p.dirty=true
  for _,button in ipairs(p.children) do if button.text=='Cancel' then button.scripts.OnClick() end end
  assert(p.enabled==ORIGINAL and not p.dirty,'Cancel failed to restore consent')
  assert(AzerMindSettingsFrame.activePage==p and p:IsShown(),'Cancel changed category')
  p.options.spells=true;p.dirty=true
  for _,button in ipairs(p.children) do if button.text=='Save' then button.scripts.OnClick() end end
  assert(not p.dirty and AzerMindSettingsFrame.activePage==p and p:IsShown(),'Save changed category')
  local options,enabled=AzerMind.ContextSettings()
  assert(options.spells and enabled==ORIGINAL,'Save changed consent or lost values')
 `);
});

test('guide list edits and removals remain drafts until Save, and Cancel restores exact variants',t=>{
 const vm=boot(t);
 vm.run(`
  ORIGINAL='https://example.com/build?variant=affliction%2C7'
  AzerMind.SaveGuideReferences(ORIGINAL)
  AzerMind.ShowGuideSettings();P=AzerMindGuideSettings
  assert(P.rows[1].input:GetText()==ORIGINAL,'existing link changed')
  assert(P.rows[1].mouseEnabled,'row surface cannot receive clicks')
  P.rows[1].input:ClearFocus();P.rows[1].scripts.OnMouseDown()
  assert(P.rows[1].input:HasFocus(),'row click did not focus its link')
  P.add.scripts.OnClick()
  P.rows[2].input:SetText('https://second.example/guide')
  P.rows[2].input.scripts.OnTextChanged(P.rows[2].input)
  assert(AzerMind.GuideReferences()==ORIGINAL,'editing saved without confirmation')
  P.rows[1].remove.scripts.OnClick()
  assert(P.rows[1].input:GetText()=='https://second.example/guide','removal did not reindex')
  assert(AzerMind.GuideReferences()==ORIGINAL,'removal changed persisted data')
  P.cancel.scripts.OnClick()
  assert(#P.entries==1 and P.rows[1].input:GetText()==ORIGINAL and not P.dirty,'Cancel lost original variant')
  P.add.scripts.OnClick();P.rows[2].input:SetText('https://second.example/guide')
  P.rows[2].input.scripts.OnTextChanged(P.rows[2].input);P.save.scripts.OnClick()
  assert(AzerMind.GuideReferences()==ORIGINAL..string.char(10)..'https://second.example/guide','Save lost order or variant')
  P.rows[2].remove.scripts.OnClick();P.rows[1].remove.scripts.OnClick();P.save.scripts.OnClick()
  assert(AzerMind.GuideReferences()=='' and #P.entries==1,'empty list is not usable')
 `);
});

test('guide row validation shows errors and blocks invalid, duplicate and oversized lists',t=>{
 const vm=boot(t);
 vm.run(`
  AzerMind.ShowGuideSettings();P=AzerMindGuideSettings
  function EDIT(index,value) P.rows[index].input:SetText(value);P.rows[index].input.scripts.OnTextChanged(P.rows[index].input) end
  EDIT(1,'javascript:bad')
  assert(P.rows[1].error and not P.save:IsEnabled(),'invalid URL not marked')
  EDIT(1,'https://example.com/guide');P.add.scripts.OnClick();EDIT(2,'https://example.com/guide')
  assert(P.rows[2].error=='This guide is already in the list.' and not P.save:IsEnabled(),'duplicate not marked')
  EDIT(2,'https://example.com/guide?variant=other')
  assert(not P.rows[2].error and P.save:IsEnabled(),'distinct variant treated as duplicate')
  EDIT(2,'https://example.com/'..string.rep('a',2048))
  assert(P.rows[2].error and not P.save:IsEnabled(),'oversized link was silently truncated')
  P.cancel.scripts.OnClick()
  for i=1,5 do
   if i>1 then P.add.scripts.OnClick() end
   EDIT(i,'https://example.com/'..i..string.rep('a',1900))
  end
  assert(not P.save:IsEnabled() and P.status:GetText():find('8000',1,true),'combined byte budget not enforced')
  assert(AzerMind.GuideReferences()=='','invalid draft changed saved guides')
 `);
});

test('guide plus/minus stays bounded at twenty rows and keyboard focus scrolls with the row',t=>{
 const vm=boot(t);
 vm.run(`
  local urls={};for i=1,20 do urls[i]='https://example.com/guide-'..i end
  AzerMind.SaveGuideReferences(table.concat(urls,string.char(10)))
  AzerMind.ShowGuideSettings();P=AzerMindGuideSettings
  assert(#P.entries==20 and not P.add:IsEnabled(),'twenty row limit missing')
  P.add.scripts.OnClick();assert(#P.entries==20,'plus exceeded maximum')
  P.rows[20].remove.scripts.OnClick()
  assert(#P.entries==19 and P.add:IsEnabled(),'minus did not free a row')
  P.rows[19].input.scripts.OnEnterPressed()
  assert(#P.entries==20 and P.rows[20].input:HasFocus(),'Enter did not focus a new row')
  assert(P.scroll:GetVerticalScroll()>0,'new row stays outside viewport')
  IsShiftKeyDown=function() return true end
  P.rows[20].input.scripts.OnTabPressed()
  assert(P.rows[19].input:HasFocus(),'Shift-Tab did not move backward')
  IsShiftKeyDown=function() return false end
  P.rows[19].input.scripts.OnTabPressed()
  assert(P.rows[20].input:HasFocus(),'Tab did not move forward')
  P.rows[20].input.scripts.OnEscapePressed(P.rows[20].input)
  assert(not P.rows[20].input:HasFocus(),'Escape kept input captured')
  assert(#P.rows==20,'row controls grew without bound')
 `);
});

test('guide rows align readable text and controls without a heavy card or an empty scrollbar',t=>{
 const vm=boot(t);
 vm.run(fs.readFileSync(path.join(__dirname,'helpers/grimoire-geometry.lua'),'utf8'));
 vm.run(`
  AzerMind.SaveGuideReferences('https://mobalytics.gg/wow-forever/classes/warlock-leveling-guide')
  AzerMind.ShowGuideSettings();local p=AzerMindGuideSettings;local row=p.rows[1]
  assert(not row.grimoireFrame,'row still has its own heavy outer frame')
  assert(row.title.justifyH=='LEFT' and row.input.justifyH=='LEFT','guide text not left-aligned')
  assert(row.input.font.size>=16,'URL text too small')
  local title,field,remove=GR_Bounds(row.title),GR_Bounds(row.inputWell),GR_Bounds(row.remove)
  assert(title.left==field.left and title.bottom>field.top,'guide title not aligned above field')
  assert(field.top==remove.top and field.height==remove.height,'remove action not aligned with field')
  assert(remove.left-field.right>=12,'input and remove button touch')
  assert(not p.scroll.ScrollBar:IsShown(),'single guide has an unnecessary empty scrollbar')
  for i=2,4 do p.add.scripts.OnClick() end
  assert(p.scroll.ScrollBar:IsShown(),'overflow lost its scrollbar')
  for i=4,2,-1 do p.rows[i].remove.scripts.OnClick() end
  assert(not p.scroll.ScrollBar:IsShown(),'scrollbar remains after removing overflow')
 `);
});

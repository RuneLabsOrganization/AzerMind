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

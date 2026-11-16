-- General settings shell. AI options are scoped to defaults or one conversation.
local panel
local Theme = AzerMindTheme
local settingsShell
local settingsPages = {}
local function SettingsMessage(page, message)
    page.message = message
    if page.Refresh and page == panel then page.Refresh()
    elseif page.status then page.status:SetText(message) end
end
local function CanOpenSettings(key)
    local active = settingsShell and settingsShell.activePage
    if active and active.settingsKey ~= key and (active.dirty or active.pending) then
        SettingsMessage(active, active.pending and "Saving settings... wait for confirmation." or "Save or cancel your changes before switching settings.")
        settingsShell:Show(); settingsShell.dimmer:Show(); active:Show()
        return false
    end
    return true
end
local function CloseSettings()
    if not settingsShell then return end
    local active = settingsShell.activePage
    if active and active.pending then SettingsMessage(active,"Saving settings... wait for confirmation."); return end
    settingsShell:Hide()
end
local function EnsureSettingsShell()
    if settingsShell then return settingsShell end
    local p = CreateFrame("Frame", "AzerMindSettingsFrame", AzerMindFrame, "BackdropTemplate")
    settingsShell = p
    p:SetSize(910,620); p:SetPoint("CENTER",UIParent,"CENTER")
    p:SetFrameStrata("FULLSCREEN_DIALOG"); p:SetFrameLevel(AzerMindFrame:GetFrameLevel()+100)
    p:EnableMouse(true); p:SetClampedToScreen(true)
    Theme.Panel(p,"frame-window",32); Theme.Material(p,"leather"); Theme.FitDialog(p)
    p.title = Theme.Label(p,"Settings",28,-28,"GameFontNormalLarge")
    local dimmer = CreateFrame("Frame","AzerMindSettingsDimmer",AzerMindFrame)
    dimmer:SetAllPoints(UIParent); dimmer:SetFrameStrata("FULLSCREEN_DIALOG")
    dimmer:SetFrameLevel(p:GetFrameLevel()-10); dimmer:EnableMouse(true); dimmer:EnableMouseWheel(true)
    if dimmer.SetIgnoreParentAlpha then dimmer:SetIgnoreParentAlpha(true) end
    local shade=dimmer:CreateTexture(nil,"BACKGROUND"); shade:SetAllPoints(); shade:SetColorTexture(0,0,0,0.62)
    dimmer:SetScript("OnMouseDown",CloseSettings); dimmer:SetScript("OnMouseWheel",function() end)
    p.dimmer=dimmer
    local close=Theme.Button(p,"X",30,30,CloseSettings)
    close:SetPoint("TOPRIGHT",p,"TOPRIGHT",-26,-24); p.close=close
    Theme.Tooltip(close,"Close settings","Escape also closes this window. Unsaved changes require confirmation.")
    local side=CreateFrame("Frame",nil,p,"BackdropTemplate")
    side:SetSize(188,476); side:SetPoint("TOPLEFT",p,"TOPLEFT",28,-88)
    Theme.Panel(side,"frame-panel",12); p.navigation=side
    local categories={
        {"ai","AI Config",function() AzerMind.CodexPrompt() end},
        {"guides","Guide references",function() AzerMind.ShowGuideSettings() end},
        {"character","Character",function() AzerMind.ShowContextSettings() end},
        {"notifications","Notifications",function() AzerMind.ShowNotificationSettings() end},
        {"keys","Key bindings",function() AzerMind.ShowKeySettings() end},
        {"appearance","Appearance",function() AzerMind.ShowAppearanceSettings() end},
        {"credits","About & credits",function() AzerMind.ShowCreditsSettings() end},
    }
    p.navButtons={}
    for i,category in ipairs(categories) do
        local key,label,action=category[1],category[2],category[3]
        local b=Theme.Button(side,label,164,38,function() if CanOpenSettings(key) then action() end end,"LEFT")
        b:SetPoint("TOPLEFT",side,"TOPLEFT",12,-16-(i-1)*48)
        b.caption:SetWordWrap(false); p.navButtons[key]=b
    end
    local divider=p:CreateTexture(nil,"BORDER"); divider:SetColorTexture(0.48,0.35,0.19,0.65)
    divider:SetPoint("BOTTOMLEFT",p,"BOTTOMLEFT",238,96); divider:SetPoint("BOTTOMRIGHT",p,"BOTTOMRIGHT",-30,96); divider:SetHeight(1)
    Theme.ProtectChanges(p,function() return p.activePage and p.activePage.dirty end,function()
        if p.activePage then p.activePage.dirty=false end
    end)
    p:HookScript("OnShow",function() dimmer:Show() end)
    p:HookScript("OnHide",function()
        if p:IsShown() then return end
        if p.activePage and p.activePage.pending and AzerMindFrame:IsShown() then
            p:Show(); SettingsMessage(p.activePage,"Saving settings... wait for confirmation."); return
        end
        local active=p.activePage
        if active and active.ClearFocus then active.ClearFocus()
        elseif active and active.input then active.input:ClearFocus() end
        if active and active.customModel then active.customModel:ClearFocus() end
        Theme.CloseChoiceMenu(); dimmer:Hide()
    end)
    table.insert(UISpecialFrames,"AzerMindSettingsFrame")
    p:Hide(); dimmer:Hide()
    return p
end
local function PrepareSettingsPage(page,key)
    local shell=EnsureSettingsShell()
    page:SetParent(shell); page:SetSize(640,510); page:ClearAllPoints()
    page:SetPoint("TOPLEFT",shell,"TOPLEFT",238,-72)
    page:SetFrameStrata("FULLSCREEN_DIALOG"); page:SetFrameLevel(shell:GetFrameLevel()+5)
    page:SetAlpha(1); page.settingsKey=key; settingsPages[key]=page
    page:Hide()
    page:HookScript("OnHide",function(self)
        if self:IsShown() or shell.switching or shell.activePage~=self or not shell:IsShown() then return end
        if self.dirty or self.pending then self:Show(); CloseSettings(); return end
        shell.activePage=nil
        if key=="ai" then CloseSettings() else AzerMind.CodexPrompt() end
    end)
end
local function ShowSettingsPage(page)
    if not CanOpenSettings(page.settingsKey) then return end
    local shell=EnsureSettingsShell()
    Theme.CloseChoiceMenu()
    GameTooltip:Hide()
    shell.switching=true
    for _,other in pairs(settingsPages) do if other~=page then other:Hide() end end
    shell.activePage=page; shell.switching=false
    for key,button in pairs(shell.navButtons) do Theme.Selected(button,key==page.settingsKey) end
    shell:Show(); shell.dimmer:Show(); page:Show()
end
local function Label(parent, text, x, y, template)
	if parent.settingsKey and template=="GameFontNormalLarge" then x,y=24,-12 end
	return Theme.Label(parent, text, x, y, template)
end
local function Button(parent, text, width, x, y, action)
	if text=="X" or text=="Close" then return end
	local b = Theme.Button(parent, text, width, 26, action)
	local footer={Cancel=368,Save=480,["Save changes"]=480,Preview=24,["Copy credits"]=24,["Reset to 100%"]=24}
	if footer[text] then x,y=footer[text],-466; b:SetHeight(32); b.settingsFooter=true end
	if text=="Save" or text=="Save changes" then Theme.Primary(b) end
	b:SetPoint("TOPLEFT", parent, "TOPLEFT", x, y)
	return b
end
local function Solid(frame, token)
	return Theme.Surface(frame, token)
end

-- A selected tab stays bright; disabled-looking buttons do not indicate scope.
local function Flat(parent, text, width, x, y, action)
	local b = Theme.Button(parent, text, width, 28, action, "LEFT")
	b:SetPoint("TOPLEFT", parent, "TOPLEFT", x, y)
	return b
end

function AzerMind.CodexPrompt(id)
    if not CanOpenSettings("ai") then return end
	local chat = AzerMind.SettingsState(id)
	if not chat then return end
	if not panel then
		local p = CreateFrame("Frame", "AzerMindGeneralSettings", AzerMindFrame, "BackdropTemplate")
		panel = p
		PrepareSettingsPage(p,"ai")
		Label(p, "AI Config", 24, -12, "GameFontNormalLarge")
		Label(p, "Applies to", 24, -48)
		p.scope = "conversation"
		local Refresh, Load
		local function Changed() p.dirty = true; p.message = "Unsaved changes"; Refresh() end
		local defaultsButton = Flat(p, "All conversations", 278, 24, -70, function()
			if p.dirty or p.pending then p.message = "Save or cancel your changes before switching scope."; Refresh(); return end
			p.scope = "default"; Load()
		end)
		local conversationButton = Flat(p, "This conversation", 278, 310, -70, function()
			if p.dirty or p.pending then p.message = "Save or cancel your changes before switching scope."; Refresh(); return end
			p.scope = "conversation"; Load()
		end)
		p.scopeHelp = Label(p, "", 24, -110); p.scopeHelp:SetWidth(572); p.scopeHelp:SetJustifyH("LEFT")
		p.inheritButton = CreateFrame("CheckButton", nil, p, "UICheckButtonTemplate")
		p.inheritButton:SetSize(24, 24); p.inheritButton:SetPoint("TOPLEFT", 24, -142)
		p.inheritButton:SetScript("OnClick", function() if not p.pending then p.inherit = not p.inherit; Changed() end end)
		p.inheritLabel = Label(p, "Use default settings", 54, -148)
        p.customize = Button(p,"Customize this conversation",240,24,-168,function() if not p.pending then p.inherit=false; Changed() end end)
		p.agentLabel = Label(p, "Agent", 24, -206)
		p.modelLabel = Label(p, "Model", 24, -276)
		p.reasonLabel = Label(p, "Reasoning effort", 24, -346)
        local dropdown={Hide=function() Theme.CloseChoiceMenu() end}
        local function Select(button, values, key)
            if p.pending or (p.scope=="conversation" and p.inherit) then return end
            local choices={}; for _,value in ipairs(values) do choices[#choices+1]={key=value[1],label=value[2]} end
            Theme.ShowChoiceMenu(button,choices,p[key],function(option)
                if option.key=="__custom" then
                    p.customModel:SetText(p.modelValue); p.customModel:Show(); p.customModel:SetFocus(); return
                end
                p[key]=option.key
                if key=="modelValue" then p.reasoningValue="" end
                if key=="agentValue" then
                    local chat=AzerMind.SettingsState(p.chatId)
                    local saved=AzerMind.ChatAISettings(chat,p.agentValue)
                    if p.scope=="default" then local _,_,_,_,_,meta=AzerMind.SettingsState(p.chatId,p.agentValue); saved=meta end
                    p.modelValue=saved.model or ""; p.reasoningValue=saved.reasoning or ""
                end
                Changed()
            end,key=="modelValue" and "Search models" or "Search options")
        end
		p.agent = Flat(p, "", 572, 24, -228, function()
			local _, _, _, agents = AzerMind.SettingsState(p.chatId, p.agentValue)
			local values = {}; for _, agent in ipairs(agents) do table.insert(values, {agent, agent}) end
			Select(p.agent, values, "agentValue")
		end)
		p.model = Flat(p, "", 572, 24, -298, function()
			local _, _, models = AzerMind.SettingsState(p.chatId, p.agentValue)
			local values = {{"", "Automatic (CLI configuration)"}}
            for _,m in ipairs(models) do table.insert(values,{m.model,(m.name or m.model)..(m.name and m.name~=m.model and (" ["..m.model.."]") or "")}) end
            table.insert(values,{"__custom","Advanced: custom model ID..."})
			Select(p.model, values, "modelValue")
		end)
		p.customModel = CreateFrame("EditBox", nil, p, "InputBoxTemplate")
		p.customModel:SetSize(564, 32); p.customModel:SetPoint("TOPLEFT", 28, -298)
		p.customModel:SetAutoFocus(false); p.customModel:SetMaxLetters(200); p.customModel:Hide()
		p.customModel:SetFrameLevel(p:GetFrameLevel() + 25)
		p.customModel:SetScript("OnTextChanged", function() if p.customModel:IsShown() then Changed() end end)
		p.customModel:SetScript("OnEnterPressed", function(self)
			p.modelValue = self:GetText():match("^%s*(.-)%s*$"); p.reasoningValue = ""
			self:ClearFocus(); self:Hide(); Changed()
		end)
		p.customModel:SetScript("OnEscapePressed", function(self) self:ClearFocus(); self:Hide() end)
		p.reasoning = Flat(p, "", 572, 24, -368, function()
            local efforts=AzerMind.ModelEfforts(p.agentValue,p.modelValue)
			local descriptions = {none="Disabled",minimal="Minimal",low="Low - faster", medium="Medium - balanced", high="High - deeper analysis", xhigh="Extra high", max="Maximum", ultra="Ultra"}
			local values = {{"", "Default (CLI configuration)"}}
			for _, effort in ipairs(efforts) do table.insert(values, {effort, descriptions[effort] or effort}) end
			Select(p.reasoning, values, "reasoningValue")
		end)
		for _, field in ipairs({p.agent, p.model, p.reasoning}) do field:SetHeight(34); Theme.SelectorAppearance(field) end
        p.catalogInfo=Label(p,"",24,-430); p.catalogInfo:SetWidth(572); p.catalogInfo:SetJustifyH("LEFT"); p.catalogInfo:SetWordWrap(false)
        p.refreshModels=Button(p,"Refresh",110,486,-266,function()
            local _,message=AzerMind.RefreshModelChoices(); p.message=message; Refresh()
        end)
        Theme.Tooltip(p.refreshModels,"Refresh model choices",function() return p.catalogInfo:GetText()..". Refreshes choices without changing your saved settings." end)
        Theme.Tooltip(p.model,function() return "Model: "..(p.effectiveModel~="" and p.effectiveModel or "Automatic") end)
        p.reasonHelp=Label(p,"",24,-368); p.reasonHelp:SetWidth(572); p.reasonHelp:SetJustifyH("LEFT")
		p.status = Label(p, "", 24, -408); p.status:SetWidth(572); p.status:SetJustifyH("LEFT")
		p.save = Button(p, "Save changes", 138, 480, -422, function()
			if p.customModel:IsShown() then
				p.modelValue = p.customModel:GetText():match("^%s*(.-)%s*$"); p.reasoningValue = ""; p.dirty = true
			end
			if p.pending or not p.dirty then return end
			local ok, message = AzerMind.SaveAISettings(p.chatId, p.scope, {agent=p.agentValue, model=p.modelValue, reasoning=p.reasoningValue, inherit=p.inherit})
			p.message = message
			if ok then p.dirty = false; p.pending = p.scope == "default" end
			Refresh()
		end)
		p.cancel=Button(p, "Cancel", 100, 370, -422, function() if not p.pending then p.Load(); p.message="Changes canceled."; p.Refresh() end end)
		Button(p, "X", 26, 594, -16, function() if not p.pending then p.dirty=false; dropdown:Hide(); p:Hide() end end)
		Refresh = function()
			local _, defaults = AzerMind.SettingsState(p.chatId, p.agentValue)
			local inherited = p.scope == "conversation" and p.inherit
			local agent = inherited and defaults.agent or p.agentValue
			local model = inherited and defaults.model or p.modelValue
			local reasoning = inherited and defaults.reasoning or p.reasoningValue
			p.inheritButton:SetShown(p.scope == "conversation" and not inherited)
            p.customize:SetShown(p.scope=="conversation" and inherited)
            p.customize:SetEnabled(not p.pending)
			p.inheritLabel:SetShown(p.scope == "conversation")
            p.inheritLabel:SetText(inherited and "Using defaults" or "Use defaults for this conversation")
			p.inheritButton:SetEnabled(not p.pending)
			p.inheritButton:SetChecked(p.inherit)
			p.scopeHelp:SetText(p.scope == "default" and "All conversations following defaults; used by new chats." or ("This conversation: " .. ((AzerMind.SettingsState(p.chatId, p.agentValue)).name or "current conversation")))
			p.agent:SetText(agent)
			p.effectiveModel=model
			local modelCaption=model
			local _,_,catalog=AzerMind.SettingsState(p.chatId,agent)
			for _,entry in ipairs(catalog) do if entry.model==model then modelCaption=entry.name or model; break end end
			p.model:SetText(model ~= "" and modelCaption or "Automatic (CLI configuration)")
			p.reasoning:SetText(reasoning ~= "" and reasoning or "Automatic (CLI configuration)")
			local _, _, _, _, _, meta = AzerMind.SettingsState(p.chatId, agent)
			p.model:Show(); p.modelLabel:Show()
            local efforts,known=AzerMind.ModelEfforts(agent,model)
            local reasoningSupported=known and #efforts>0
            if not inherited and p.reasoningValue~="" then
                local valid=false; for _,effort in ipairs(efforts) do if effort==p.reasoningValue then valid=true end end
                if not known or not valid then p.reasoningValue=""; p.dirty=true; p.message="Unsupported reasoning cleared to Automatic."; p.reasoning:SetText("Automatic (CLI configuration)") end
            end
            p.reasonHelp:SetText(known and "Reasoning is fixed for this model." or "Reasoning options are not known for this model. Using Automatic.")
            if inherited and not known and reasoning~="" then p.reasonHelp:SetText("Inherited reasoning: "..reasoning.." (capabilities not reported).") end
            p.reasonHelp:SetShown(not reasoningSupported)
            local catalogState={stale=" - cached; refresh failed",cached=" - cached",["not-installed"]=" - CLI not found",unavailable=" - no local catalog"}
            p.catalogInfo:Hide()
            p.catalogInfo:SetText((meta.catalogSource or "Connect to load model choices.")..(catalogState[meta.catalogStatus] or ""))
            p.refreshModels:SetEnabled(not p.pending)
			p.reasoning:SetShown(reasoningSupported); p.reasonLabel:SetShown(reasoningSupported)
			p.agent:SetEnabled(not inherited and not p.pending)
			p.model:SetEnabled(not inherited and not p.pending); p.reasoning:SetEnabled(not inherited and not p.pending)
			p.save:SetEnabled(p.dirty and not p.pending)
			p.cancel:SetEnabled(not p.pending)
			for _, tab in ipairs({defaultsButton, conversationButton}) do
				tab:SetEnabled(not p.pending)
				local selected = (tab == defaultsButton) == (p.scope == "default")
				Theme.Selected(tab, selected)
			end
			for _, field in ipairs({p.agent, p.model, p.reasoning}) do field:SetAlpha(1); Theme.Color(field.caption,"SetTextColor","text")
                if field.grimoireFrame then for _,piece in ipairs(field.grimoireFrame.parts) do piece:SetVertexColor(1,1,1,1) end end end
			p.status:SetText(p.message or (inherited and "Using defaults. Choose Customize to change this conversation." or "Changes apply to your next message."))
		end
		Load = function()
			local c, defaults, _, _, pending = AzerMind.SettingsState(p.chatId, p.agentValue)
			p.pending = pending ~= nil; p.dirty = false; p.message = nil
			local options = AzerMind.ChatAISettings(c)
			p.inherit = (c.agent or "") == "" and (options.model or "") == "" and (options.reasoning or "") == ""
			p.agentValue = p.scope == "default" and defaults.agent or ((c.agent and c.agent ~= "") and c.agent or defaults.agent)
			p.modelValue = p.scope == "default" and defaults.model or (options.model or "")
			p.reasoningValue = p.scope == "default" and defaults.reasoning or (options.reasoning or "")
			p.customModel:Hide(); dropdown:Hide(); Refresh()
		end
		p.Refresh, p.Load = Refresh, Load
		AzerMind.AttachSettingsPanel(p)
	end
	if panel.dirty then
		if panel.chatId~=chat.id then panel.message="Save or cancel changes before switching conversations."; panel.Refresh() end
		ShowSettingsPage(panel); return
	end
	panel.chatId = chat.id; panel.Load(); ShowSettingsPage(panel)
end


local contextPanel
function AzerMind.ShowContextSettings()
    if not CanOpenSettings("character") then return end
	if not contextPanel then
		local p = CreateFrame("Frame", "AzerMindContextSettings", AzerMindFrame, "BackdropTemplate")
		contextPanel = p
		p:SetSize(600, 470); p:SetPoint("CENTER", AzerMindFrame, "CENTER")
		p:SetFrameStrata("FULLSCREEN_DIALOG"); p:SetFrameLevel(AzerMindFrame:GetFrameLevel() + 100)
		p:EnableMouse(true); p:SetClampedToScreen(true); PrepareSettingsPage(p,"character")
		Label(p, "Share character context", 20, -18, "GameFontNormalLarge")
		Label(p, "Choose what accompanies future messages in all conversations.", 20, -46)
		local function Check(text, x, y, action)
			local b = CreateFrame("CheckButton", nil, p, "UICheckButtonTemplate")
			b:SetSize(24, 24); b:SetPoint("TOPLEFT", x, y)
			b.caption=Label(p, text, x + 28, y - 5)
			b:SetScript("OnClick", action); return b
		end
		local Refresh
		p.enabledBox = Check("Enable automatic sharing", 20, -76, function(self) p.enabled = self:GetChecked() == true; p.dirty=true; Refresh() end)
		p.checks = {}
		Refresh = function()
			p.enabledBox:SetChecked(p.enabled)
			for key, box in pairs(p.checks) do
				box:SetChecked(p.options[key]); box:SetEnabled(p.enabled)
				box.caption:SetAlpha(p.enabled and 1 or 0.5)
			end
			for _,button in ipairs(p.presets or {}) do button:SetEnabled(p.enabled) end
			if p.save then p.save:SetEnabled(p.dirty) end
			if p.status and p.dirty then p.status:SetText("Unsaved changes.") end
		end
		p.presets={
			Button(p, "Basic", 104, 24, -112, function() p.options = AzerMind.ContextOptions("basic"); p.dirty=true; Refresh() end),
			Button(p, "All available", 130, 136, -112, function() p.options = AzerMind.ContextOptions("full"); p.dirty=true; Refresh() end),
			Button(p, "Clear selection", 140, 274, -112, function() p.options = {}; p.dirty=true; Refresh() end)
		}
		for i, field in ipairs(AzerMind.ContextCategories) do
			local key, caption = field[1], field[2]
			local column, row = (i - 1) % 2, math.floor((i - 1) / 2)
			p.checks[key] = Check(caption, 20 + column * 280, -156 - row * 28, function(self)
				p.options[key] = self:GetChecked() == true
				p.dirty=true
				Refresh()
			end)
		end
		local help = Label(p, "The preview marks missing data and size limits. Previously shared data may still appear in your chat history.", 20, -338)
		help:SetWidth(560); help:SetJustifyH("LEFT")
		p.status=Label(p,"",24,-404); p.status:SetWidth(572); p.status:SetJustifyH("LEFT")
		Button(p, "Preview", 110, 20, -414, function()
			AzerMind.ShowCopy(p.enabled and AzerMind.GameContext(p.options) or "Automatic character context sharing is off.")
			AzerMindCopy:SetFrameLevel(p:GetFrameLevel() + 20)
		end)
		Button(p, "Cancel", 100, 346, -414, function()
            p.options,p.enabled=AzerMind.ContextSettings(); p.dirty=false; Refresh(); p.status:SetText("Changes canceled.")
        end)
		p.save=Button(p, "Save", 120, 454, -414, function()
			AzerMind.SaveContextSettings(p.options, p.enabled); p.dirty=false; Refresh(); p.status:SetText("Saved. Applies to future messages.")
		end)
		Button(p, "X", 26, 554, -14, function() p:Hide() end)
		p.Refresh = Refresh
	end
	local p = contextPanel
	if p.dirty then ShowSettingsPage(p); return end
	p.options, p.enabled = AzerMind.ContextSettings()
	p.Refresh(); ShowSettingsPage(p)
end

-- Editable rows; persistence remains newline-delimited for existing installations.
local guidePanel
function AzerMind.ShowGuideSettings()
    if not CanOpenSettings("guides") then return end
    if not guidePanel then
        local p=CreateFrame("Frame","AzerMindGuideSettings",AzerMindFrame,"BackdropTemplate")
        guidePanel=p; p:EnableMouse(true); PrepareSettingsPage(p,"guides")
        Label(p,"Guide references",24,-12,"GameFontNormalLarge")
        local help=Label(p,"Add the guides you want the AI to consult. Applies to all conversations.",24,-54)
        help:SetWidth(572); help:SetJustifyH("LEFT")
        p.count=Label(p,"",24,-94)
        p.count:SetJustifyH("LEFT"); Theme.Color(p.count,"SetTextColor","muted")
        local scroll=CreateFrame("ScrollFrame","AzerMindGuideListScroll",p,"UIPanelScrollFrameTemplate")
        scroll:SetPoint("TOPLEFT",p,"TOPLEFT",24,-132); scroll:SetSize(540,260)
        Theme.Scrollbar(scroll); p.scroll=scroll
        local content=CreateFrame("Frame",nil,scroll); content:SetSize(532,1); scroll:SetScrollChild(content)
        p.rows={}; p.entries={}; p.focusedIndex=1
        local rowStep=84
        p.status=Label(p,"",24,-408); p.status:SetWidth(572); p.status:SetJustifyH("LEFT")
        local function TrimURL(value) return (value or ""):match("^%s*(.-)%s*$") end
        local function Serialize()
            local result={}
            for _,value in ipairs(p.entries) do local url=TrimURL(value); if url~="" then result[#result+1]=url end end
            return table.concat(result,"\n")
        end
        function p.ClearFocus()
            for _,row in ipairs(p.rows) do row.input:ClearFocus(); Theme.Focus(row.inputWell,false) end
        end
        local function FocusRow(index)
            local row=p.rows[index]; if not row or not row:IsShown() then return end
            p.ClearFocus(); p.focusedIndex=index; row.input:SetFocus(); Theme.Focus(row.inputWell,true)
            local top,bottom=(index-1)*rowStep,index*rowStep
            local offset=scroll:GetVerticalScroll()
            if top<offset then offset=top elseif bottom>offset+scroll:GetHeight() then offset=bottom-scroll:GetHeight() end
            scroll:SetVerticalScroll(math.max(0,math.min(offset,math.max(0,content:GetHeight()-scroll:GetHeight()))))
        end
        local function Validate()
            local seen,count,invalid={},0,false
            for index,value in ipairs(p.entries) do
                local url=TrimURL(value); local row=p.rows[index]; local message
                if url~="" then
                    count=count+1
                    if not url:match("^https?://[^%s/]+") or url:find("[%s|%c]") then message="Use a complete http:// or https:// link."
                    elseif #url>2048 then message="This link exceeds 2048 bytes."
                    elseif seen[url] then message="This guide is already in the list." end
                    seen[url]=true
                end
                local host=url:match("^https?://([^/%?#]+)")
                row.title:SetText(index..". "..(host or "New guide"))
                row.feedback:SetText(message or (url=="" and "Paste a guide link" or ""))
                Theme.Color(row.feedback,"SetTextColor",message and "accent" or "muted")
                row.error=message; invalid=invalid or message~=nil
            end
            local text=Serialize()
            if #text>8000 then invalid=true end
            p.dirty=text~=AzerMind.GuideReferences()
            p.count:SetText(count.." / 20 guide links")
            p.add:SetEnabled(#p.entries<20)
            p.save:SetEnabled(p.dirty and not invalid)
            if #text>8000 then p.status:SetText("The guide list exceeds 8000 bytes. Shorten or remove a link.")
            elseif invalid then p.status:SetText("Check the marked links before saving.")
            elseif p.dirty then p.status:SetText("Unsaved changes. Save to apply; Cancel restores your saved guides.")
            elseif count==0 then p.status:SetText("Paste a link in the first row. Use + Add guide for more.")
            else p.status:SetText("Used for future messages. Previously shared links remain in chat history.") end
        end
        local Render
        local function AddGuide()
            if #p.entries>=20 then return end
            GameTooltip:Hide()
            p.entries[#p.entries+1]=""; Render(); FocusRow(#p.entries)
        end
        Render=function()
            p.rendering=true
            for index,value in ipairs(p.entries) do
                local row=p.rows[index]
                if not row then
                    row=CreateFrame("Frame",nil,content); row:SetSize(532,80); row:EnableMouse(true)
                    row.title=Label(row,"",0,-2); row.title:SetWidth(532); row.title:SetWordWrap(false); row.title:SetJustifyH("LEFT")
                    Theme.Type(row.title,"heading")
                    row.inputWell=CreateFrame("Frame",nil,row,"BackdropTemplate")
                    row.inputWell:SetSize(480,36); row.inputWell:SetPoint("TOPLEFT",row,"TOPLEFT",0,-24)
                    Theme.Panel(row.inputWell,"button-secondary",6); row.inputWell:EnableMouse(true)
                    row.inputWell:SetScript("OnMouseDown",function() FocusRow(row.index) end)
                    row.input=CreateFrame("EditBox",nil,row.inputWell)
                    row.input:SetSize(480,36); row.input:SetPoint("TOPLEFT",row.inputWell,"TOPLEFT",0,0)
                    row.input:SetTextInsets(10,10,0,0); row.input:SetJustifyH("LEFT")
                    row.input:SetAutoFocus(false); row.input:SetMultiLine(false); row.input:SetMaxLetters(10000)
                    Theme.Type(row.input,"paperBody"); Theme.Color(row.input,"SetTextColor","text")
                    row.feedback=Label(row,"",0,-64); row.feedback:SetWidth(532); row.feedback:SetWordWrap(false); row.feedback:SetJustifyH("LEFT")
                    local divider=row:CreateTexture(nil,"BORDER"); divider:SetHeight(1)
                    divider:SetPoint("BOTTOMLEFT",row,"BOTTOMLEFT",0,0); divider:SetPoint("BOTTOMRIGHT",row,"BOTTOMRIGHT",0,0)
                    local border=Theme.colors.border; divider:SetColorTexture(border[1],border[2],border[3],0.35)
                    row.remove=Theme.Button(row,"-",36,36,function()
                        local removed=row.index
                        GameTooltip:Hide()
                        p.ClearFocus(); table.remove(p.entries,removed)
                        if #p.entries==0 then p.entries[1]="" end
                        Render(); FocusRow(math.min(removed,#p.entries))
                    end)
                    row.remove:SetPoint("TOPLEFT",row,"TOPLEFT",496,-24)
                    Theme.Tooltip(row.remove,function() return "Remove guide "..row.index end,"Removal takes effect when you save. Cancel restores saved links.")
                    Theme.Tooltip(row.input,function() return TrimURL(row.input:GetText()) end)
                    row:SetScript("OnMouseDown",function() FocusRow(row.index) end)
                    row.input:SetScript("OnTextChanged",function(self)
                        if p.rendering then return end
                        self=self or row.input; p.entries[row.index]=self:GetText() or ""; Validate()
                    end)
                    row.input:SetScript("OnEditFocusGained",function() p.focusedIndex=row.index; Theme.Focus(row.inputWell,true) end)
                    row.input:SetScript("OnEditFocusLost",function() Theme.Focus(row.inputWell,false) end)
                    row.input:SetScript("OnEscapePressed",function(self) (self or row.input):ClearFocus() end)
                    row.input:SetScript("OnEnterPressed",function()
                        if row.index<#p.entries then FocusRow(row.index+1) else AddGuide() end
                    end)
                    row.input:SetScript("OnTabPressed",function()
                        local step=IsShiftKeyDown and IsShiftKeyDown() and -1 or 1
                        FocusRow(((row.index-1+step)%#p.entries)+1)
                    end)
                    p.rows[index]=row
                end
                row.index=index; row:ClearAllPoints(); row:SetPoint("TOPLEFT",content,"TOPLEFT",0,-(index-1)*rowStep)
                row.input:SetText(value); row:Show()
            end
            for index=#p.entries+1,#p.rows do p.rows[index].input:ClearFocus(); p.rows[index]:Hide() end
            content:SetHeight(math.max(1,#p.entries*rowStep))
            scroll:UpdateScrollChildRect()
            local overflow=content:GetHeight()>scroll:GetHeight()
            local bar=scroll.ScrollBar or _G["AzerMindGuideListScrollScrollBar"]
            if bar then
                bar:SetShown(overflow)
                for _,key in ipairs({"ScrollUpButton","ScrollDownButton"}) do
                    local arrow=bar[key] or scroll[key] or _G["AzerMindGuideListScrollScrollBar"..key]
                    if arrow then arrow:SetShown(overflow) end
                end
            end
            if not overflow then scroll:SetVerticalScroll(0) end
            p.input=p.rows[1].input -- first-row alias for existing focus/diagnostic callers
            p.rendering=false; Validate()
        end
        p.add=Button(p,"+ Add guide",154,442,-82,AddGuide); p.add:SetHeight(32)
        p.cancel=Button(p,"Cancel",100,368,-466,function() p.Load(); p.status:SetText("Changes canceled.") end)
        p.save=Button(p,"Save",120,480,-466,function()
            local ok,message=AzerMind.SaveGuideReferences(Serialize())
            if ok then p.Load(); p.status:SetText("Saved. Applies to future messages in all conversations.")
            else p.status:SetText(message) end
        end)
        function p.Load()
            p.ClearFocus(); p.entries={}
            for url in AzerMind.GuideReferences():gmatch("[^\n]+") do p.entries[#p.entries+1]=url end
            if #p.entries==0 then p.entries[1]="" end
            p.focusedIndex=1; scroll:SetVerticalScroll(0); Render()
        end
        p:SetScript("OnHide",p.ClearFocus)
        p.FocusRow=FocusRow
    end
    local p=guidePanel
    if not p.dirty then p.Load() end
    ShowSettingsPage(p); p.FocusRow(p.focusedIndex or 1)
end

function AzerMind.ShowNotificationSettings()
    if not CanOpenSettings("notifications") then return end
    local p=AzerMindNotificationSettings
    if not p then
        p=CreateFrame("Frame","AzerMindNotificationSettings",AzerMindFrame,"BackdropTemplate")
        p:SetSize(520,280); p:SetPoint("CENTER"); p:SetFrameStrata("FULLSCREEN_DIALOG"); p:SetFrameLevel(AzerMindFrame:GetFrameLevel()+120)
        p:EnableMouse(true);  PrepareSettingsPage(p,"notifications");
        Label(p,"Reply notifications",110,-24,"GameFontNormalLarge")
        Label(p,"Applies to all conversations. Changes are saved immediately.",20,-70)
        local options={{key=0,label="None"},{key=3081,label="Whisper"},{key=18019,label="Battle.net toast"},{key=3175,label="Map ping"},{key=8960,label="Ready check"},{key=7355,label="Tutorial popup"}}
        p.sound=Theme.Selector(p,"Sound",320,options,AzerMind.ReplySound(),function(o) AzerMind.ReplySound(o.key) end)
        p.sound:SetPoint("TOPLEFT",20,-110)
        Button(p,"Preview",120,360,-110,function() local sound=AzerMind.ReplySound(); if sound~=0 then pcall(PlaySound,sound,"Master") end end)
        Button(p,"Close",100,380,-226,function() p:Hide() end)
        p.options=options
    end
    for _,o in ipairs(p.options) do if o.key==AzerMind.ReplySound() then p.sound:SetValue(o.key,o.label) end end
    ShowSettingsPage(p)
end

function AzerMind.ShowKeySettings()
    if not CanOpenSettings("keys") then return end
 local p=AzerMindKeySettings
 if not p then
  p=CreateFrame("Frame","AzerMindKeySettings",AzerMindFrame); p:SetSize(560,340); p:SetPoint("CENTER")
  p:SetFrameStrata("FULLSCREEN_DIALOG"); p:SetFrameLevel(AzerMindFrame:GetFrameLevel()+120); p:EnableMouse(true)
   PrepareSettingsPage(p,"keys");
  Label(p,"Chat key bindings",110,-24,"GameFontNormalLarge")
  Label(p,"Enter a key combination, for example CTRL-SHIFT-C.",20,-68)
  p.status=Label(p,"",20,-230); p.status:SetWidth(510); p.status:SetJustifyH("LEFT")
  for i,spec in ipairs({{"Open chat","AzerMindOpenChatKeyButton",function() AzerMind.Toggle(true) end},{"New chat","AzerMindNewChatKeyButton",function() AzerMind.NewChat() end}}) do
   local input=Theme.SearchField(p,nil,300,spec[1]); input:SetPoint("TOPLEFT",24,-116-(i-1)*70); input:SetMaxLetters(50)
   Button(p,"Bind",100,436,-116-(i-1)*70,function()
    if InCombatLockdown() then p.status:SetText("Key bindings can be changed outside combat."); return end
    local key=(input:GetText() or ""):upper():gsub("%s","")
    if key=="" or not key:match("^[%w%-]+$") then p.status:SetText("Enter a valid key combination."); return end
    local command="CLICK "..spec[2]..":LeftButton"
    local existing=GetBindingAction and GetBindingAction(key)
    if existing and existing~="" and existing~=command then p.status:SetText(key.." is already bound to "..existing..". Choose another key."); return end
    local first,second
    if GetBindingKey then first,second=GetBindingKey(command) end
    if SetBinding(key,command)==false then p.status:SetText("The client rejected this key combination."); return end
    if first and first~=key then SetBinding(first) end
    if second and second~=key then SetBinding(second) end
    SaveBindings(GetCurrentBindingSet()); p.status:SetText(spec[1].." bound to "..key..". Opening the chat focuses the message field.")
   end)
   local current=GetBindingKey and GetBindingKey("CLICK "..spec[2]..":LeftButton"); input:SetText(current or "")
  end
  Button(p,"Close",100,436,-286,function() p:Hide() end)
 end
 ShowSettingsPage(p)
end

local appearancePanel
function AzerMind.ShowAppearanceSettings()
    if not CanOpenSettings("appearance") then return end
 if not appearancePanel then
  local p=CreateFrame("Frame","AzerMindAppearanceSettings",AzerMindFrame,"BackdropTemplate")
  appearancePanel=p;p:SetSize(500,250);p:SetPoint("CENTER",AzerMindFrame,"CENTER")
  p:SetFrameStrata("FULLSCREEN_DIALOG");p:SetFrameLevel(AzerMindFrame:GetFrameLevel()+100)
  p:EnableMouse(true);PrepareSettingsPage(p,"appearance")
  Label(p,"Appearance",24,-24,"GameFontNormalLarge")
  Label(p,"Addon opacity",24,-68)
  Label(p,"Applies to the chat, mini bar and addon windows.",24,-94)
  p.percent=Label(p,"",540,-68)
  p.slider=CreateFrame("Slider","AzerMindOpacitySlider",p,"OptionsSliderTemplate")
  p.slider:SetPoint("TOPLEFT",p,"TOPLEFT",32,-126);p.slider:SetSize(548,20)
  p.slider:SetMinMaxValues(35,100);p.slider:SetValueStep(1)
  for _,entry in ipairs({{"Low","35%"},{"High","100%"},{"Text",""}}) do
   local label=_G["AzerMindOpacitySlider"..entry[1]]; if label then label:SetText(entry[2]) end
  end
  if p.slider.SetObeyStepOnDrag then p.slider:SetObeyStepOnDrag(true) end
  p.slider:SetScript("OnValueChanged",function(_,value)
   if p.loading then return end
   value=math.floor(value+0.5);AzerMind.SetOpacity(value/100);p.percent:SetText(value.."%")
  end)
  Label(p,"Changes apply immediately and are saved.",24,-164)
  Button(p,"Reset to 100%",140,180,-202,function() p.slider:SetValue(100) end)
  Button(p,"Close",80,394,-202,function() p:Hide() end)
 end
 local p=appearancePanel;p.loading=true;p.slider:SetValue(AzerMind.GetOpacity()*100)
 p.percent:SetText(math.floor(AzerMind.GetOpacity()*100+0.5).."%");p.loading=false;ShowSettingsPage(p)
end


function AzerMind.ShowCreditsSettings()
    if not CanOpenSettings("credits") then return end
    local p = AzerMindCreditsSettings
    if not p then
        p = CreateFrame("Frame", "AzerMindCreditsSettings", AzerMindFrame, "BackdropTemplate")
        p:SetSize(600, 470); p:SetPoint("CENTER", AzerMindFrame, "CENTER")
        p:SetFrameStrata("FULLSCREEN_DIALOG"); p:SetFrameLevel(AzerMindFrame:GetFrameLevel()+100)
        p:EnableMouse(true); PrepareSettingsPage(p,"credits")
        Label(p, "About & credits", 24, -24, "GameFontNormalLarge")
        local intro = Label(p, "AzerMind is made possible by these projects and contributors.", 24, -58)
        intro:SetWidth(548); intro:SetJustifyH("LEFT")
        local entries = {
            {"AzerMind contributors", "Fork development, character tools, catalogs and interface.", "https://github.com/mateusmaaia/AzerMind-wow"},
            {"chelinho139 / wow-ai", "Original addon, conversations, agent bridge, transport, maps and macros. Original MIT copyright retained.", "https://github.com/chelinho139/wow-ai"},
            {"AHledger", "Optional auction prices from player scans. Bundled snapshots are labeled; prices are not guaranteed sales.", "Prices from AHledger (ahledger.com)"},
            {"GearQuest Forever", "Optional active equipment hunts for the current character. AzerMind reads recommendations without changing them."},
            {"ForeverDungeonJournal", "Optional dungeon loot sources and bundled factual instance references. Authors and contributors retain credit."},
            {"Questie and QuestieDB teams and contributors", "Bundled item, quest, NPC and object records. Source licenses, historical notices and provenance accompany the data.", "https://github.com/Questie/QuestieDB"},
            {"0xInuarashi / wow-forever-codex", "Pixel-output research and Forever file-loading findings used by the original project.", "https://github.com/0xinuarashi/wow-forever-codex"},
            {"Gethe / wow-ui-source", "Blizzard UI source mirror used to check client APIs.", "https://github.com/Gethe/wow-ui-source"},
            {"Blizzard Entertainment", "World of Warcraft, game assets and announced raid information belong to their respective owners."},
        }
        local scroll = CreateFrame("ScrollFrame", "AzerMindCreditsScroll", p, "UIPanelScrollFrameTemplate")
        scroll:SetPoint("TOPLEFT", p, "TOPLEFT", 24, -92); scroll:SetSize(520, 300)
        local content = CreateFrame("Frame", nil, scroll); content:SetWidth(516)
        scroll:SetScrollChild(content); Theme.Scrollbar(scroll)
        local y, text = 0, {}
        for _,entry in ipairs(entries) do
            local title = Label(content, entry[1], 0, -y, "GameFontNormal")
            title:SetWidth(510); title:SetJustifyH("LEFT")
            local description = Label(content, entry[2] .. (entry[3] and ("\n" .. entry[3]) or ""), 0, -y-24)
            description:SetWidth(510); description:SetJustifyH("LEFT"); description:SetWordWrap(true)
            y = y + 24 + description:GetStringHeight() + 22
            text[#text+1] = table.concat(entry, "\n")
        end
        content:SetHeight(y)
        p.creditText = table.concat(text, "\n\n") .. "\n\nFull licenses and source provenance are included with the addon. This is an independent community project."
        local footnote = Label(p, "UI revision 2026-11-15. Full licenses and source notices accompany the addon.", 24, -406)
        footnote:SetWidth(548); footnote:SetJustifyH("LEFT")
        Button(p, "Copy credits", 130, 300, -430, function() AzerMind.ShowCopy(p.creditText) end)
        Button(p, "Close", 110, 446, -430, function() p:Hide() end)
    end
    ShowSettingsPage(p)
end

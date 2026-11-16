-- Inventory browsing is read-only. Closed-bank contents are explicitly cached.
local T, picker, bankOpen = AzerMindTheme, nil, false
local function Read(fn, ...)
	if type(fn) ~= "function" then return nil end
	local ok, a,b,c,d,e,f,g,h,i,j,k,l,m,n = pcall(fn, ...)
	if ok then return a,b,c,d,e,f,g,h,i,j,k,l,m,n end
	return nil
end
local function Cache()
	AzerMindItemsDB = AzerMindItemsDB or {}
	local key = (Read(GetRealmName) or "") .. ":" .. (Read(UnitName, "player") or "")
	AzerMindItemsDB[key] = AzerMindItemsDB[key] or {}
	return AzerMindItemsDB[key]
end
local function ScanBank()
	if not bankOpen then return end
	local api, items = C_Container or {}, {}
	for bag = -1, 11 do
		if bag == -1 or bag >= 5 then
			for slot = 1, tonumber(Read(api.GetContainerNumSlots or GetContainerNumSlots, bag)) or 0 do
				local link = Read(api.GetContainerItemLink or GetContainerItemLink, bag, slot)
				if link then
					local info = Read(api.GetContainerItemInfo, bag, slot)
					local _, count = Read(GetContainerItemInfo, bag, slot)
					items[#items+1] = {link=link, count=type(info)=="table" and info.stackCount or count or 1}
				end
			end
		end
	end
	local cache = Cache(); cache.bank = items; cache.updated = date("%Y-%m-%d %H:%M")
end
function AzerMind.OwnedItems()
	local result, grouped = {}, {}
	local function Add(link, count, location)
		if type(link) ~= "string" then return end
		local payload, label = link:match("|H(item:[^|]+)|h(.-)|h")
		if not payload then return end
		local key = location .. ":" .. payload
		if grouped[key] then grouped[key].count = grouped[key].count + (count or 1); return end
		local query = C_Item and C_Item.GetItemInfo or GetItemInfo
		local name, _, quality, _, level, kind, subtype, _, slot, icon, _, classID, subclassID = Read(query, link)
		local item = {key=key, link=AzerMind.NormalizeItemLink(link), name=name or AzerMind.PlainText(label),
			id=tonumber(payload:match("item:(%d+)")), count=count or 1, location=location, quality=quality,
			level=level, kind=kind or "", subtype=subtype or "", slot=slot or "", icon=icon, classID=classID, subclassID=subclassID}
		grouped[key]=item; result[#result+1]=item
	end
	for slot=1,19 do Add(Read(GetInventoryItemLink, "player", slot), 1, "Equipped") end
	local api = C_Container or {}
	for bag=0,4 do
		for slot=1,tonumber(Read(api.GetContainerNumSlots or GetContainerNumSlots, bag)) or 0 do
			local link = Read(api.GetContainerItemLink or GetContainerItemLink, bag, slot)
			local info = Read(api.GetContainerItemInfo, bag, slot)
			local _, count = Read(GetContainerItemInfo, bag, slot)
			Add(link, type(info)=="table" and info.stackCount or count or 1, "Bags")
		end
	end
	for _, item in ipairs(Cache().bank or {}) do Add(item.link, item.count, "Bank") end
	table.sort(result, function(a,b) if a.name == b.name then return a.key < b.key end; return a.name < b.name end)
	return result
end
local categoryFamilies = {
    {"all","All items"},{"class:4","Armor",4},{"class:2","Weapons",2},
    {"accessories","Accessories"},{"class:0","Consumables",0},{"class:7","Materials",7},
    {"class:9","Recipes",9},{"class:1","Containers",1},{"class:3","Gems",3},
    {"class:5","Reagents",5},{"class:6","Projectiles",6},{"class:11","Quivers",11},
    {"class:13","Keys",13},{"class:12","Quest items",12},{"class:15","Miscellaneous",15},
    {"other","Unknown / Other"},
}
local armorNames = {[0]="Miscellaneous armor",[1]="Cloth",[2]="Leather",[3]="Mail",[4]="Plate",[6]="Shields"}
local weaponNames = {[0]="One-handed axes",[1]="Two-handed axes",[2]="Bows",[3]="Guns",[4]="One-handed maces",
    [5]="Two-handed maces",[6]="Polearms",[7]="One-handed swords",[8]="Two-handed swords",[10]="Staves",
    [13]="Fist weapons",[15]="Daggers",[16]="Thrown",[18]="Crossbows",[19]="Wands",[20]="Fishing poles"}
local accessorySlots = {INVTYPE_NECK=true, INVTYPE_FINGER=true, INVTYPE_TRINKET=true, INVTYPE_CLOAK=true}
local function Category(item, key)
    if key=="all" then return true end
    if key=="accessories" then return accessorySlots[item.slot] or false end
    if key=="other" then return type(item.classID)~="number" or item.classID<0 end
    local class=tonumber(key:match("^class:(%d+)$"))
    if class then return item.classID==class end
    local parent,sub=key:match("^sub:(%d+):(%d+)$")
    if parent then return item.classID==tonumber(parent) and item.subclassID==tonumber(sub) end
    parent=tonumber(key:match("^unknown:(%d+)$"))
    return parent and item.classID==parent and type(item.subclassID)~="number" or false
end
local function ItemCategoryTree(items)
    local roots,nodes,observed={},{},{}
    for _,family in ipairs(categoryFamilies) do
        local node={key=family[1],label=family[2],classID=family[3],children={}}
        roots[#roots+1]=node; nodes[node.key]=node
    end
    for _,item in ipairs(items) do
        if type(item.classID)=="number" and item.classID>=0 then
            local key="class:"..item.classID
            if not nodes[key] then
                local node={key=key,label=Read(GetItemClassInfo,item.classID) or ("Item class "..item.classID),classID=item.classID,children={}}
                roots[#roots+1]=node; nodes[key]=node
            end
            observed[item.classID]=observed[item.classID] or {}
            observed[item.classID][type(item.subclassID)=="number" and item.subclassID or "unknown"]=item.subtype or ""
        end
    end
    for class,children in pairs(observed) do
        local parent=nodes["class:"..class]
        for sub,name in pairs(children) do
            local key=sub=="unknown" and ("unknown:"..class) or ("sub:"..class..":"..sub)
            local fallback=class==4 and armorNames[sub] or class==2 and weaponNames[sub]
            local label=sub=="unknown" and "Unknown subtype" or Read(GetItemSubClassInfo,class,sub)
                or (type(name)=="string" and name~="" and name) or fallback or ("Subclass "..sub)
            local node={key=key,label=label,parent=parent.key,children={}}
            parent.children[#parent.children+1]=node; nodes[key]=node
        end
        table.sort(parent.children,function(a,b) if a.label==b.label then return a.key<b.key end; return a.label<b.label end)
    end
    return roots,nodes
end
local function ItemSearch(text)
    text=(text or ""):match("^%s*(.-)%s*$"):lower()
    local id=text:match("item:(%d+)") or text:match("^(%d+)$")
    local words={}; if not id then for word in text:gmatch("%S+") do words[#words+1]=word end end
    return {id=tonumber(id),words=words}
end
local function SearchMatches(item, search)
    if search.id then return item.id==search.id end
    local text=table.concat({item.name or "",item.kind or "",item.subtype or ""}," "):lower()
    for _,word in ipairs(search.words) do if not text:find(word,1,true) then return false end end
    return true
end
function AzerMind.ShowItemPicker()
	local chat = AzerMind.SettingsState(); if not chat then return end
	if not picker then
		local p = CreateFrame("Frame", "AzerMindItemPicker", AzerMindFrame)
		picker=p; p:SetSize(920,660); p:SetPoint("CENTER"); p:SetClampedToScreen(true)
		p:SetFrameStrata("FULLSCREEN_DIALOG"); p:SetFrameLevel(AzerMindFrame:GetFrameLevel()+110); p:EnableMouse(true)
		if p.SetIgnoreParentAlpha then p:SetIgnoreParentAlpha(true) end
		T.Panel(p,"frame-window",32); T.Material(p,"leather"); T.FitDialog(p); table.insert(UISpecialFrames,"AzerMindItemPicker")
		T.Label(p,"Items",32,-26,T.fonts.title)
		local function Button(label,w,x,y,action)
			local b=T.Button(p,label,w,30,action); b:SetPoint("TOPLEFT",x,y); return b
		end
		Button("X",30,858,-24,function() p:Hide() end)
		p.mode="owned"
		local tab=Button("My items",128,32,-66,function() p.mode="owned"; p.location="All"; p.onlySelected=false; p.Refresh() end)
		local db=Button("Item database",148,172,-66,function() p.mode="catalog"; p.location="All"; p.onlySelected=false; p.Refresh() end)
		p.version=T.Label(p,"",340,-74); p.version:SetWidth(548); p.version:SetJustifyH("LEFT")
		local search=T.SearchField(p,nil,632,"Search by name, type, or item ID")
		search:SetPoint("TOPLEFT",256,-128); p.search=search
		search:SetMaxLetters(100); search:SetScript("OnEscapePressed",function(self) self:ClearFocus() end)
		p.rows,p.selected,p.category,p.location,p.rarity,p.level,p.slot={}, {},"all","All",-1,false,"All"
		local scroll=CreateFrame("ScrollFrame","AzerMindItemPickerScroll",p,"UIPanelScrollFrameTemplate")
		-- Reserve the entire selection toolbar below the viewport. Previously
		-- the viewport extended through it, so rows covered its buttons.
		scroll:SetPoint("TOPLEFT",p,"TOPLEFT",256,-256); scroll:SetPoint("BOTTOMRIGHT",p,"BOTTOMRIGHT",-54,164); T.Scrollbar(scroll); p.scroll=scroll
		if scroll.SetClipsChildren then scroll:SetClipsChildren(true) end
		local content=CreateFrame("Frame",nil,scroll); content:SetSize(610,1); scroll:SetScrollChild(content)
		p.summary=T.Label(p,"",32,-600); p.summary:SetWidth(530); p.summary:SetJustifyH("LEFT")
		p.bank=T.Label(p,"",32,-622); p.bank:SetWidth(530); p.bank:SetJustifyH("LEFT"); p.bank:SetWordWrap(false)
		p.page=1
		p.pageLabel=T.Label(p,"",368,-516); p.pageLabel:SetWidth(414); p.pageLabel:SetJustifyH("CENTER")
		local categoryButtons, locationButtons={},{}
        p.expanded={}; p.categoryRows={}; p.categoryCounts={}
        local RenderTree
		local function Render()
			if p.rendering or p.changingFilters then return end
			p.rendering=true
			local visible, total, hidden=p.visible or {},0,0
            local filter=search:GetText() or ""
            local baseQuery=table.concat({p.mode,filter,p.location,p.rarity,tostring(p.minLevel),tostring(p.maxLevel),p.slot},":")
            if baseQuery~=p.baseQuery then
                p.baseQuery=baseQuery; p.query=nil; p.baseMatches={}; p.categoryCounts={}; p.unknownMetadata=0
                p.filterPasses=(p.filterPasses or 0)+1
                local term=ItemSearch(filter)
                for _,item in ipairs(p.items or {}) do
                    if p.slot~="All" then AzerMind.EnrichCatalogItem(item) end
                    local location=p.location=="All" or item.location==p.location or (p.mode=="catalog" and p.ownership[item.id] and p.ownership[item.id][p.location])
                    local unknown=(p.rarity~=-1 and item.quality==nil) or (p.slot~="All" and (item.slot or "")=="") or ((p.minLevel or p.maxLevel) and item.level==nil)
                    if location and SearchMatches(item,term) then
                        if unknown then p.unknownMetadata=p.unknownMetadata+1 end
                        local match=(p.rarity==-1 or item.quality==p.rarity)
                            and (not p.minLevel or (item.level and item.level>=p.minLevel))
                            and (not p.maxLevel or (item.level and item.level<=p.maxLevel))
                            and (p.slot=="All" or item.slot==p.slot)
                        if match then
                            p.baseMatches[#p.baseMatches+1]=item
                            local keys={"all",type(item.classID)=="number" and ("class:"..item.classID) or "other"}
                            if type(item.classID)=="number" then keys[#keys+1]=type(item.subclassID)=="number" and ("sub:"..item.classID..":"..item.subclassID) or ("unknown:"..item.classID) end
                            if accessorySlots[item.slot] then keys[#keys+1]="accessories" end
                            for _,key in ipairs(keys) do p.categoryCounts[key]=(p.categoryCounts[key] or 0)+1 end
                        end
                    end
                end
            end
            local query=baseQuery..":"..p.category..":"..tostring(p.onlySelected)
            if query~=p.query then
                p.query=query; p.page=1; visible={}; p.matchKeys={}; scroll:SetVerticalScroll(0)
                for _,item in ipairs(p.baseMatches) do
                    if Category(item,p.category) then
                        p.matchKeys[item.key]=true
                        if not p.onlySelected or p.selected[item.key] then visible[#visible+1]=item end
                    end
                end
            end
			for key in pairs(p.selected) do total=total+1; if not p.matchKeys[key] then hidden=hidden+1 end end
			p.visible=visible
			local pages=math.max(1,math.ceil(#visible/25)); p.page=math.min(p.page,pages)
			local pageStart=(p.page-1)*25
			local pageCount=math.min(25,#visible-pageStart)
			local first=math.floor(scroll:GetVerticalScroll()/48)+1
			local used=0
			for i=first,math.min(pageCount,first+5) do
				local item=visible[pageStart+i]; used=used+1
				if p.mode=="catalog" then AzerMind.EnrichCatalogItem(item) end
				local row=p.rows[used]
				if not row then
					row=CreateFrame("CheckButton",nil,content,"UICheckButtonTemplate"); row:SetSize(24,24)
					row:SetHitRectInsets(0,-570,0,-20)
					row.icon=row:CreateTexture(nil,"ARTWORK"); row.icon:SetSize(30,30); row.icon:SetPoint("TOPLEFT",30,-2)
					row.title=T.Label(row,"",68,-1); row.title:SetWidth(520); row.title:SetJustifyH("LEFT"); row.title:SetWordWrap(false)
					row.detail=T.Label(row,"",68,-22); row.detail:SetWidth(520); row.detail:SetJustifyH("LEFT"); row.detail:SetWordWrap(false); T.Color(row.detail,"SetTextColor","muted")
					row:SetScript("OnClick",function(self) p.selected[self.item.key]=self:GetChecked() or nil; if p.onlySelected then p.query=nil end; Render() end)
					local function ShowItemTooltip(self)
                        self.auctionExpanded = IsShiftKeyDown and IsShiftKeyDown() or false
						GameTooltip:SetOwner(self,"ANCHOR_RIGHT"); GameTooltip:ClearLines()
						Read(GameTooltip.SetHyperlink,GameTooltip,self.item.link)
						if p.mode=="catalog" then GameTooltip:AddLine(AzerMind.CatalogItemDetails(self.item.id),0.8,0.85,0.85,true) end
						local auction = AzerMind.AuctionItemDetails and AzerMind.AuctionItemDetails(self.item.id, not self.auctionExpanded)
						if auction then GameTooltip:AddLine(auction,0.8,0.85,0.85,true) end
						GameTooltip:Show()
                    end
                    row:SetScript("OnEnter",function(self)
                        ShowItemTooltip(self)
                        self:SetScript("OnUpdate",function(owner)
                            local expanded = IsShiftKeyDown and IsShiftKeyDown() or false
                            if GameTooltip:GetOwner() == owner and expanded ~= owner.auctionExpanded then ShowItemTooltip(owner) end
                        end)
                    end)
					row:RegisterForClicks("LeftButtonUp","RightButtonUp")
                    local select=row:GetScript("OnClick")
                    row:SetScript("OnClick",function(self,button) if button=="RightButton" then AzerMind.ShowSourceBrowser("item",self.item.id) else select(self) end end)
                    row:SetScript("OnLeave",function(self) self:SetScript("OnUpdate",nil); GameTooltip:Hide() end); p.rows[used]=row
				end
				row:ClearAllPoints(); row:SetPoint("TOPLEFT",0,-(i-1)*48)
				row.item=item; row:SetChecked(p.selected[item.key] or false); row.icon:SetTexture(item.icon or 134400)
				row.title:SetText(item.link..(item.count>1 and (" ×"..item.count) or ""))
				local slotName=_G[item.slot] or item.slot
				row.detail:SetText((item.subtype~="" and item.subtype or "Uncached type")..(slotName~="" and (" · "..slotName) or "").." · "..item.location)
				row:Show()
			end
			for i=used+1,#p.rows do p.rows[i]:Hide() end
			content:SetHeight(math.max(1,pageCount*48))
			p.pageLabel:SetText("Page "..p.page.." / "..pages)
			p.previous:SetEnabled(p.page>1); p.next:SetEnabled(p.page<pages)
			p.summary:SetText(total.." selected · "..hidden.." hidden by filters · "..#visible.." shown"..(#visible==0 and " · No matching items" or ""))
			p.insert:SetText("Add to message ("..total..")"); p.insert:SetEnabled(total>0 and total<=100)
			local cache=Cache(); p.bank:SetText(cache.updated and ("Bank · "..(bankOpen and "Live" or "Last updated "..cache.updated)) or "Open your bank to load its items.")
            if RenderTree then RenderTree() end
            if p.UpdateFilterSummary then p.UpdateFilterSummary() end
            if p.empty then
                p.empty:SetShown(#visible==0); p.searchAll:SetShown(#visible==0 and p.category~="all")
                p.emptyReset:SetShown(#visible==0)
                p.empty:SetText(p.catalogError or ((p.minLevel and p.maxLevel and p.minLevel>p.maxLevel) and "Min level must not exceed Max level." or (p.unknownMetadata>0 and "No matching items with known metadata. Some item data is unavailable." or "No matching items. Broaden your category or clear a filter.")))
            end
			p.locationSelector.value=p.location; p.locationSelector:SetText((p.mode=="catalog" and "Owned in: " or "Location: ")..p.location)
			if p.mode=="catalog" then
				p.bank:SetText(total>100 and "Choose up to 100 items per attachment." or "Rarity and slot filters require client-cached data.")
			end
            if p.unknownMetadata>0 then p.bank:SetText(p.unknownMetadata.." items have missing data for the current filters.") end
            if total>100 then p.bank:SetText("Choose up to 100 items per attachment.") end
            if p.minLevel and p.maxLevel and p.minLevel>p.maxLevel then p.bank:SetText("Required level: Min must not exceed Max.") end
            p.rendering=false
		end
		p.previous=Button("Previous",92,256,-508,function() p.page=math.max(1,p.page-1); scroll:SetVerticalScroll(0); Render() end)
		p.next=Button("Next",92,796,-508,function() p.page=p.page+1; scroll:SetVerticalScroll(0); Render() end)
		scroll:HookScript("OnVerticalScroll",function() if p.insert then Render() end end)
        function p.SetLocation(value)
            p.location=value; p.query=nil; scroll:SetVerticalScroll(0); Render()
        end
        p.locationSelector=T.Selector(p,"Location",200,{{key="All",label="All"},{key="Equipped",label="Equipped"},{key="Bags",label="Bags"},{key="Bank",label="Bank"}},"All",function(option) p.SetLocation(option.key) end)
        p.locationSelector:SetPoint("TOPLEFT",32,-128); p.locationSelector:SetValue("All","All")
        T.Label(p,"Location / owned in",32,-106)

		local categoryScroll=CreateFrame("ScrollFrame","AzerMindItemCategoryScroll",p,"UIPanelScrollFrameTemplate")
        categoryScroll:SetPoint("TOPLEFT",32,-196); categoryScroll:SetSize(188,300); T.Scrollbar(categoryScroll)
        T.Label(p,"Categories",32,-172)
        local categoryContent=CreateFrame("Frame",nil,categoryScroll); categoryContent:SetSize(188,1); categoryScroll:SetScrollChild(categoryContent)
        p.categoryScroll=categoryScroll
        function p.SelectCategory(key)
            if not p.categoryNodes[key] then return end
            p.category=key; p.expanded[key]=true; p.query=nil; scroll:SetVerticalScroll(0); GameTooltip:Hide(); Render()
        end
        function p.ToggleCategory(key) p.expanded[key]=not p.expanded[key]; GameTooltip:Hide(); RenderTree() end
        RenderTree=function()
            local entries={}
            for _,node in ipairs(p.categoryRoots or {}) do
                entries[#entries+1]=node
                if p.expanded[node.key] then for _,child in ipairs(node.children) do entries[#entries+1]=child end end
            end
            for index,node in ipairs(entries) do
                local row=p.categoryRows[index]
                if not row then
                    row=CreateFrame("Frame",nil,categoryContent); row:SetSize(188,34)
                    row.select=T.Button(row,"",164,32,function() p.SelectCategory(row.node.key) end,"LEFT")
                    row.select:SetPoint("TOPLEFT",24,0); row.select.caption:SetWidth(114)
                    row.count=T.Label(row,"",150,-9); row.count:SetWidth(36); row.count:SetJustifyH("RIGHT")
                    T.Tooltip(row.select,function() return row.node.label end,"Counts follow the source and other filters.")
                    row.toggle=T.Button(row,"+",22,32,function() p.ToggleCategory(row.node.key) end)
                    row.toggle:SetPoint("TOPLEFT",0,0); p.categoryRows[index]=row
                end
                row.node=node; row:ClearAllPoints(); row:SetPoint("TOPLEFT",0,-(index-1)*34)
                row.select:SetText((node.parent and "  " or "")..node.label); row.count:SetText(tostring(p.categoryCounts[node.key] or 0))
                row.toggle:SetShown(#node.children>0); row.toggle:SetText(p.expanded[node.key] and "-" or "+")
                T.Selected(row.select,p.category==node.key); row:Show()
            end
            for index=#entries+1,#p.categoryRows do p.categoryRows[index]:Hide() end
            p.treeEntries=entries; categoryContent:SetHeight(math.max(1,#entries*34)); categoryScroll:UpdateScrollChildRect()
            if p.breadcrumb then p.breadcrumb:SetText(p.categoryNodes[p.category] and (p.categoryNodes[p.category].parent and ((p.categoryNodes[p.categoryNodes[p.category].parent].label).." > ") or "")..p.categoryNodes[p.category].label or "All items") end
        end

		local rarityOptions={{key=-1,label="All"}}
        for value=0,7 do rarityOptions[#rarityOptions+1]={key=value,label=_G["ITEM_QUALITY"..value.."_DESC"] or tostring(value)} end
        p.raritySelector=T.Selector(p,"Rarity",140,rarityOptions,-1,function(option) p.rarity=option.key; Render() end)
        p.raritySelector:SetPoint("TOPLEFT",256,-180); p.raritySelector:SetValue(-1,"All")
		T.Label(p,"Min level",580,-160); T.Label(p,"Max level",656,-160)
        local function LevelInput(name,x,field)
            local input=T.SearchField(p,name,64,"")
            input.label:SetText(""); input:SetPoint("TOPLEFT",x,-180); input:SetNumeric(true); input:SetMaxLetters(3)
            input:SetScript("OnTextChanged",function(self) p[field]=tonumber(self:GetText()); Render() end)
            return input
        end
        p.minInput=LevelInput("AzerMindItemMinLevel",580,"minLevel")
        p.maxInput=LevelInput("AzerMindItemMaxLevel",656,"maxLevel")
        p.breadcrumb=T.Label(p,"All items",256,-230); p.breadcrumb:SetWidth(420); p.breadcrumb:SetJustifyH("LEFT")

		local slots={"All","INVTYPE_HEAD","INVTYPE_NECK","INVTYPE_SHOULDER","INVTYPE_CHEST","INVTYPE_WAIST","INVTYPE_LEGS","INVTYPE_FEET","INVTYPE_WRIST","INVTYPE_HAND","INVTYPE_FINGER","INVTYPE_TRINKET","INVTYPE_CLOAK","INVTYPE_WEAPON","INVTYPE_2HWEAPON","INVTYPE_WEAPONMAINHAND","INVTYPE_WEAPONOFFHAND","INVTYPE_HOLDABLE","INVTYPE_SHIELD","INVTYPE_RANGED","INVTYPE_RANGEDRIGHT"}
        local slotOptions={}
        for _,value in ipairs(slots) do slotOptions[#slotOptions+1]={key=value,label=_G[value] or value} end
        p.slotSelector=T.Selector(p,"Slot",156,slotOptions,"All",function(option) p.slot=option.key; Render() end)
        p.slotSelector:SetPoint("TOPLEFT",408,-180)
        function p.ClearFilter(key)
            p.changingFilters=true
            if key=="search" or key=="all" then search:SetText("") end
            if key=="category" or key=="all" then p.category="all" end
            if key=="location" or key=="all" then p.location="All" end
            if key=="rarity" or key=="all" then p.rarity=-1; p.raritySelector:SetValue(-1,"All") end
            if key=="slot" or key=="all" then p.slot="All"; p.slotSelector:SetValue("All","All") end
            if key=="level" or key=="all" then p.minLevel=nil;p.maxLevel=nil;p.minInput:SetText("");p.maxInput:SetText("") end
            if key=="all" then p.onlySelected=false;p.selectedToggle:SetValue(false) end
            p.changingFilters=false; p.query=nil; Render()
        end
        p.reset=Button("Reset filters",152,736,-180,function() p.ClearFilter("all") end)
        p.filterMenu=Button("Filters: 0",200,32,-508,function()
            T.ShowChoiceMenu(p.filterMenu,p.activeFilters or {},nil,function(option) p.ClearFilter(option.key) end,"Clear one filter")
        end)
        function p.UpdateFilterSummary()
            local active={}
            local function Add(key,label) active[#active+1]={key=key,label=label} end
            if search:GetText()~="" then Add("search","Search: "..search:GetText()) end
            if p.category~="all" then Add("category","Category: "..p.categoryNodes[p.category].label) end
            if p.location~="All" then Add("location","Location: "..p.location) end
            if p.rarity~=-1 then Add("rarity","Rarity: "..(_G["ITEM_QUALITY"..p.rarity.."_DESC"] or p.rarity)) end
            if p.slot~="All" then Add("slot","Slot: "..(_G[p.slot] or p.slot)) end
            if p.minLevel or p.maxLevel then Add("level","Level: "..(p.minLevel or "Any").." - "..(p.maxLevel or "Any")) end
            p.activeFilters=active; p.filterMenu:SetText("Filters: "..#active);p.filterMenu:SetEnabled(#active>0)
        end
        p.empty=T.Label(scroll,"",16,-26);p.empty:SetWidth(570);p.empty:SetJustifyH("LEFT");p.empty:SetWordWrap(true)
        p.searchAll=T.Button(scroll,"Search all items",180,30,function() p.ClearFilter("category") end)
        p.searchAll:SetPoint("TOPLEFT",scroll,"TOPLEFT",16,-88)
        p.emptyReset=T.Button(scroll,"Reset filters",160,30,function() p.ClearFilter("all") end)
        p.emptyReset:SetPoint("TOPLEFT",scroll,"TOPLEFT",208,-88)
        search:SetScript("OnEnterPressed",function() p.query=nil;Render() end)

		local divider=p:CreateTexture(nil,"BORDER")
		divider:SetHeight(1); divider:SetPoint("TOPLEFT",p,"TOPLEFT",32,-542); divider:SetPoint("TOPRIGHT",p,"TOPRIGHT",-32,-542)
		T.Color(divider,"SetColorTexture","border")
		Button("Select all results",144,160,-556,function() for _,item in ipairs(p.visible) do p.selected[item.key]=true end; if p.onlySelected then p.query=nil end; Render() end)
		Button("Clear selection",128,316,-556,function() p.selected={}; if p.onlySelected then p.query=nil end; Render() end)
		p.selectedToggle=T.Toggle(p,"Show selected only",210,false,function(value) p.onlySelected=value; p.query=nil; Render() end); p.selectedToggle:SetPoint("TOPLEFT",460,-556)
		Button("Select page",116,32,-556,function()
			-- Replace the selection so items hidden by the current filters cannot
			-- accidentally ride along when selecting only equipment, bags, etc.
			for i=(p.page-1)*25+1,math.min(p.page*25,#p.visible) do p.selected[p.visible[i].key]=true end
			Render()
		end)
		p.insert=Button("Add to message (0)",192,696,-600,function()
			local chosen=0; for _ in pairs(p.selected) do chosen=chosen+1 end
            if chosen==0 then return end
			if chosen>100 then p.summary:SetText("Choose up to 100 items per attachment."); return end
			local lines={"--- Selected inventory items ---", "Inventory snapshot: "..date("%Y-%m-%d %H:%M")..". Only Equipped entries are worn."}
			for _,item in ipairs(p.items) do
				if p.selected[item.key] then
					if p.mode=="catalog" then AzerMind.EnrichCatalogItem(item) end
					lines[#lines+1]=item.location..": "..item.link.." ×"..item.count..(item.location=="Bank" and (" (bank snapshot: "..(Cache().updated or "unknown")..")") or "")
					if p.mode=="catalog" then lines[#lines+1]=AzerMind.CatalogItemDetails(item.id) end
					local auction = AzerMind.AuctionItemDetails and AzerMind.AuctionItemDetails(item.id)
					if auction then lines[#lines+1]=auction end
				end
			end
			lines[#lines+1]="--- End selected inventory items ---"
			local attachment=table.concat(lines,"\n")
			if #attachment>36000 then p.summary:SetText("This attachment is too large. Choose fewer items."); return end
			if AzerMind.AppendQuestDraft(attachment,p.chatId) then p:Hide() else p.summary:SetText("Conversation changed. Reopen Items to attach this selection.") end
		end)
		T.Primary(p.insert)
		T.Tooltip(p.insert,"Add items to message","Maximum 100 items per attachment. Adds to the draft; nothing is sent yet.")
        p:SetScript("OnHide",T.CloseChoiceMenu)
		Button("Cancel",100,584,-600,function() p:Hide() end)
		search:SetScript("OnTextChanged",function()
			p.searchVersion=(p.searchVersion or 0)+1; local version=p.searchVersion
			C_Timer.After(0.15,function() if p.insert and p:IsShown() and p.searchVersion==version then scroll:SetVerticalScroll(0); Render() end end)
		end)
		p.Refresh=function(metadataOnly)
			T.SourceSelection(p,p.mode)
            p.selectedToggle:SetValue(p.onlySelected)
            p.query=nil; p.baseQuery=nil
			local err
			if p.mode=="catalog" then p.items,err=AzerMind.CatalogItemList() else p.items=AzerMind.OwnedItems() end
			p.catalogError=err
            p.items=p.items or {}
            for _,item in ipairs(p.items) do
                if item.classID==nil or (p.mode=="catalog" and (item.slot or "")=="") then
                    local _,_,_,slot,_,class,sub=Read(C_Item and C_Item.GetItemInfoInstant or GetItemInfoInstant,item.link or item.id)
                    item.classID=item.classID or class; item.subclassID=item.subclassID or sub
                    if slot and slot~="" then item.slot=slot end
                end
            end
            p.categoryRoots,p.categoryNodes=ItemCategoryTree(p.items)
            if not p.categoryNodes[p.category] then p.category="all" end
			if p.lastMode~=p.mode then scroll:SetVerticalScroll(0); p.lastMode=p.mode end
			T.Selected(tab,p.mode=="owned"); T.Selected(db,p.mode=="catalog")
			p.ownership={}
            for _,owned in ipairs(AzerMind.OwnedItems()) do
                if owned.id then p.ownership[owned.id]=p.ownership[owned.id] or {}; p.ownership[owned.id][owned.location]=true end
            end
            for _,b in pairs(locationButtons) do b:SetEnabled(true) end
			p.version:SetText(p.mode=="catalog" and (AzerMindCatalog and ("Forever snapshot · "..AzerMindCatalog.manifest.source.." "..AzerMindCatalog.manifest.version) or "Catalog unavailable") or "Equipment, bags and last bank snapshot")
            p.removedSelections=T.ReconcileSelection(p.selected,p.items,"key")
			Render()
            if p.removedSelections>0 then p.summary:SetText(p.removedSelections.." unavailable selections removed. "..p.summary:GetText()) end
			if err then p.summary:SetText(err) end
		end
	end
	if picker.chatId~=chat.id then picker.selections={}; picker.selectionSource=nil; picker.selected={} end
    picker.chatId=chat.id; picker:Show(); picker.Refresh()
end
local events=CreateFrame("Frame")
for _,event in ipairs({"BANKFRAME_OPENED","BANKFRAME_CLOSED","BAG_UPDATE_DELAYED","PLAYER_EQUIPMENT_CHANGED","GET_ITEM_INFO_RECEIVED"}) do events:RegisterEvent(event) end
events:SetScript("OnEvent",function(_,event)
	if event=="BANKFRAME_OPENED" then bankOpen=true; ScanBank()
	elseif event=="BANKFRAME_CLOSED" then bankOpen=false
	elseif event=="BAG_UPDATE_DELAYED" then ScanBank() end
	if picker and picker:IsShown() then
        if event=="GET_ITEM_INFO_RECEIVED" then
            if not picker.metadataQueued then
                picker.metadataQueued=true
                C_Timer.After(0.1,function()
                    picker.metadataQueued=nil
                    if picker:IsShown() then picker.Refresh(true) end
                end)
            end
        else picker.Refresh() end
    end
end)

-- Read-only context captured on demand for action bar and keybind questions.
local LAYOUT_WORDS = {
	bar = true, bars = true, actionbar = true, actionbars = true, barra = true, barras = true,
	hotkey = true, hotkeys = true, keybind = true, keybinds = true, keybinding = true, keybindings = true,
	atalho = true, atalhos = true, tecla = true, teclas = true,
	reorganizar = true, reorganize = true, reorganise = true, rearrange = true,
}
local ABILITY_WORDS = {
	skill = true, skills = true, spell = true, spells = true, ability = true, abilities = true,
	habilidade = true, habilidades = true, magia = true, magias = true,
}
local ORGANIZE_WORDS = { organizar = true, organize = true, organise = true, arrumar = true, ordenar = true }

local function IsLayoutQuestion(text)
	if type(text) ~= "string" then return false end
	local ability, organize = false, false
	for word in text:lower():gmatch("[%w_\128-\255]+") do
		if LAYOUT_WORDS[word] then return true end
		ability = ability or ABILITY_WORDS[word]
		organize = organize or ORGANIZE_WORDS[word]
	end
	return ability and organize
end

local function IsSecret(value)
	return type(issecretvalue) == "function" and issecretvalue(value)
end

local function Read(fn, ...)
	if type(fn) ~= "function" then return false end
	local ok, first, second, third, fourth = pcall(fn, ...)
	if not ok or IsSecret(first) or IsSecret(second) or IsSecret(third) or IsSecret(fourth) then
		return false
	end
	return true, first, second, third, fourth
end

local function Value(fn, ...)
	local _, result = Read(fn, ...)
	return result
end

local function Count(value)
	return not IsSecret(value) and type(value) == "number" and value >= 0
		and value < math.huge and value % 1 == 0
end

local function ID(value)
	return Count(value) and value > 0
end

local function Label(value)
	if IsSecret(value) or type(value) ~= "string" then return nil end
	local label = value:gsub("%c", " "):match("^%s*(.-)%s*$")
	if label ~= "" then return label end
end

local function SpellName(id)
	return Label(Value(C_Spell and C_Spell.GetSpellName, id)) or Label(Value(GetSpellInfo, id))
end

local function SpellRank(id)
	return Label(Value(C_Spell and C_Spell.GetSpellSubtext, id)) or Label(Value(GetSpellSubtext, id))
end

local function RankNumber(rank)
	-- Compare only a single reported numeric rank. An absent or ambiguous rank stays distinct.
	if not rank then return nil end
	local number, following = rank:match("(%d+)(.*)")
	if number and not following:find("%d") then return tonumber(number) end
end

local function ReadSpellbook(add)
	local book = C_SpellBook
	local modern = book and book.GetNumSpellBookSkillLines and book.GetSpellBookSkillLineInfo and book.GetSpellBookItemInfo
	local tabCount = Value(modern and book.GetNumSpellBookSkillLines or GetNumSpellTabs)
	if not Count(tabCount) then return false, false end
	local partial = tabCount > 30
	local bank = Enum and Enum.SpellBookSpellBank and Enum.SpellBookSpellBank.Player or 0
	local spellType = Enum and Enum.SpellBookItemType and Enum.SpellBookItemType.Spell or 1
	for tab = 1, math.min(tabCount, 30) do
		local offset, count, skip
		if modern then
			local info = Value(book.GetSpellBookSkillLineInfo, tab)
			if type(info) == "table" then
				offset, count, skip = info.itemIndexOffset, info.numSpellBookItems, info.offSpecID
			end
		else
			local _, _, _, bookOffset, bookCount = Read(GetSpellTabInfo, tab)
			offset, count = bookOffset, bookCount
		end
		if IsSecret(skip) or not (Count(offset) and Count(count)) then
			partial = true
		elseif not skip then
			partial = partial or count > 500
			for index = offset + 1, offset + math.min(count, 500) do
				local ok, name, rank, id
				if modern then
					local item = Value(book.GetSpellBookItemInfo, index, bank)
					ok = type(item) == "table" and not (IsSecret(item.itemType) or IsSecret(item.isOffSpec) or IsSecret(item.isPassive))
					if ok and item.itemType == spellType and not item.isOffSpec and not item.isPassive then
						local _, reportedName, reportedRank = Read(book.GetSpellBookItemName, index, bank)
						name, rank, id = Label(reportedName) or Label(item.name), reportedRank, item.spellID
						ok = ID(id)
					end
				else
					local bookType = BOOKTYPE_SPELL or "spell"
					local kind, spellID
					ok, kind, spellID = Read(GetSpellBookItemInfo, index, bookType)
					if ok and kind == "SPELL" then
						local checked, passive = Read(IsPassiveSpell, index, bookType)
						ok = checked
						if checked and not passive then
							local _, reportedName, reportedRank = Read(GetSpellBookItemName, index, bookType)
							name, rank, id = Label(reportedName), reportedRank, spellID
							ok = ID(id)
						end
					end
				end
				if not ok then
					partial = true
				elseif id then
					name = name or SpellName(id)
					if name then add(name, Label(rank) or SpellRank(id), id)
					else partial = true end
				end
			end
		end
	end
	return true, partial
end

local function AppendWithin(lines, entries, limit, partial, notice)
	local size = 0
	for _, entry in ipairs(entries) do
		if size + #entry + 1 > limit then
			partial = true
			break
		end
		lines[#lines + 1] = entry
		size = size + #entry + 1
	end
	if partial then lines[#lines + 1] = notice end
end

local function ReadLearnedAbilities(lines)
	local groups, names = {}, {}
	local readable, partial = ReadSpellbook(function(name, rank, id)
		local group = groups[name]
		if not group then
			group = {unknown = {}, seen = {}}
			groups[name] = group
			names[#names + 1] = name
		end
		local number = RankNumber(rank)
		if number then
			if not group.best or number > group.best.number then
				group.best = {number = number, rank = rank, id = id}
			end
		elseif not group.seen[id] then
			group.unknown[#group.unknown + 1] = {rank = rank, id = id}
			group.seen[id] = true
		end
	end)
	lines[#lines + 1] = "Learned active abilities (highest known rank and entries with unverified rank):"
	if not readable then
		lines[#lines + 1] = "Active spellbook unavailable."
		return
	end
	local entries = {}
	for _, name in ipairs(names) do
		local group = groups[name]
		if group.best then
			entries[#entries + 1] = name .. " (" .. group.best.rank .. ") #" .. group.best.id
		end
		for _, spell in ipairs(group.unknown) do
			if not group.best or spell.id ~= group.best.id then
				entries[#entries + 1] = name .. " (" .. (spell.rank or "rank unavailable") .. ") #" .. spell.id
			end
		end
	end
	if #entries == 0 then
		lines[#lines + 1] = partial and "Active spells unavailable in this snapshot." or "No active spells reported."
	end
	AppendWithin(lines, entries, 1800, partial, "[Partial active spellbook: some entries unavailable or omitted at size limit.]")
end

local BARS = {
	{"Main", "ActionButton", "ACTIONBUTTON"},
	{"Bottom left", "MultiBarBottomLeftButton", "MULTIACTIONBAR1BUTTON"},
	{"Bottom right", "MultiBarBottomRightButton", "MULTIACTIONBAR2BUTTON"},
	{"Right", "MultiBarRightButton", "MULTIACTIONBAR3BUTTON"},
	{"Left", "MultiBarLeftButton", "MULTIACTIONBAR4BUTTON"},
	{"Bar 5", "MultiBar5Button", "MULTIACTIONBAR5BUTTON"},
	{"Bar 6", "MultiBar6Button", "MULTIACTIONBAR6BUTTON"},
	{"Bar 7", "MultiBar7Button", "MULTIACTIONBAR7BUTTON"},
}

local function BindingKeys(command)
	local ok, first, second = Read(GetBindingKey, command)
	local bindingByKey = C_KeyBindings and C_KeyBindings.GetBindingByKey
	local keys, partial = {}, not ok
	local candidates = {Label(first), Label(second)}
	for index = 1, 2 do
		local key = candidates[index]
		if key then
			local read, current
			if bindingByKey then read, current = Read(bindingByKey, key)
			elseif GetBindingAction then read, current = Read(GetBindingAction, key, true) end
			if not (bindingByKey or GetBindingAction) or (read and current == command) then
				keys[#keys + 1] = key
			else
				partial = true
			end
		end
	end
	return keys, partial
end

local function ButtonKeys(button, buttonName, command)
	local regular, regularPartial = BindingKeys(Label(button.bindingAction) or command)
	local click, clickPartial = BindingKeys("CLICK " .. buttonName .. ":LeftButton")
	local partial = regularPartial or clickPartial
	local keys, seen = {}, {}
	for _, source in ipairs({regular, click}) do
		for _, key in ipairs(source) do
			if not seen[key] then
				keys[#keys + 1] = key
				seen[key] = true
			end
		end
	end
	if #keys > 0 then return table.concat(keys, ", "), partial end
	return partial and "bindings unavailable" or "unbound", partial
end

local function ActionLabel(slot)
	if not ID(slot) then return "action unavailable" end
	local ok, kind, id = Read(GetActionInfo or (C_ActionBar and C_ActionBar.GetActionInfo), slot)
	if not ok then return "action unavailable" end
	if kind == nil then
		local hasAction = C_ActionBar and C_ActionBar.HasAction or HasAction
		if hasAction and Value(hasAction, slot) ~= false then return "action unavailable" end
		return "empty"
	end
	kind = Label(kind)
	if not kind then return "action unavailable" end
	if kind == "spell" and ID(id) then
		return (SpellName(id) or "spell") .. " (" .. (SpellRank(id) or "rank unavailable") .. ") #" .. id
	end
	if kind == "item" and ID(id) then
		local name = Label(Value(C_Item and C_Item.GetItemInfo, id)) or Label(Value(GetItemInfo, id))
		return name and ("item " .. name .. " #" .. id) or ("item #" .. id)
	end
	return kind .. (ID(id) and (" #" .. id) or "")
end

local function ReadActionBars(lines)
	local entries, partial = {}, false
	for _, bar in ipairs(BARS) do
		for index = 1, 12 do
			local buttonName = bar[2] .. index
			local button = _G[buttonName]
			if button then
				local visible, shown = Read(button.IsVisible, button)
				if not visible then
					partial = true
				elseif shown then
					local slot = button.action
					local position = ID(slot) and (" [slot " .. slot .. "]") or " [slot unavailable]"
					local action = ActionLabel(slot)
					local keys, keysPartial = ButtonKeys(button, buttonName, bar[3] .. index)
					partial = partial or action == "action unavailable" or keysPartial
					entries[#entries + 1] = bar[1] .. " " .. index .. position .. ": " .. action .. " | " .. keys
				end
			end
		end
	end
	if #entries == 0 then lines[#lines + 1] = "Visible default action buttons unavailable." end
	AppendWithin(lines, entries, 2600, partial, "[Partial action bars: some buttons or keys unavailable or omitted at size limit.]")
end

function AzerMind.ActionBarQuestionContext(rawText)
	if not IsLayoutQuestion(rawText) then return "" end
	local options, enabled = AzerMind.ContextSettings()
	if not enabled or not options.spells then return "" end
	local lines = {
		"\n\n--- Current action bars and keybinds (question-scoped snapshot) ---",
		"Scope is visible default player buttons. Hidden or inactive pages, third-party bars, pet, vehicle and stance bars may be unobserved.",
		"Keys come from configured button bindings. Unlisted keys may be occupied by other actions or overrides. Suggest changes for the player to make.",
	}
	ReadActionBars(lines)
	ReadLearnedAbilities(lines)
	return table.concat(lines, "\n")
end

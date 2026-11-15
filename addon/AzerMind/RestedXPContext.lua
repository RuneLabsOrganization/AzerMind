-- Read runtime fields only. RestedXP's progress and condition methods may write state.
local MAX_BYTES = 3000
local MAX_TEXT_BYTES = 120
local MAX_ELEMENTS_PER_STEP = 6
local PARTIAL = "RestedXP route partial: some steps, fields or quest states could not be shown."
local questActions = { accept = "accept", complete = "complete", turnin = "turn in" }
local travelActions = { ["goto"] = "travel", waypoint = "travel", fly = "fly", fp = "learn flight path",
    hs = "hearth", home = "set hearth", zone = "enter zone" }

local function Table(value)
    return type(value) == "table" and value or nil
end

local function PositiveInteger(value)
    if type(value) == "number" and value > 0 and value <= 4294967295 and value == math.floor(value) then
        return value
    end
end

-- Keep a valid UTF-8 prefix even when byte limits fall inside a character.
local function Text(value)
    if type(value) ~= "string" then return nil, value ~= nil end
    local pos = 1
    while pos <= math.min(#value, MAX_TEXT_BYTES) do
        local first, second = value:byte(pos, pos + 1)
        local width = first < 128 and 1 or first >= 194 and first < 224 and 2
            or first >= 224 and first < 240 and 3 or first >= 240 and first <= 244 and 4
        if not width or pos + width - 1 > MAX_TEXT_BYTES or pos + width - 1 > #value then break end
        if (first == 224 and second < 160) or (first == 237 and second >= 160)
            or (first == 240 and second < 144) or (first == 244 and second >= 144) then break end
        local tail = value:sub(pos + 1, pos + width - 1)
        if tail:find("[^\128-\191]") then break end
        pos = pos + width
    end
    local text = value:sub(1, pos - 1)
    if AzerMind.PlainText then text = AzerMind.PlainText(text) end
    text = text:gsub("[%c]", " "):gsub("%s+", " "):match("^%s*(.-)%s*$")
    return text ~= "" and text or nil, pos <= #value
end

local function CurrentStep(guide, data)
    if not data then return end
    local progress = Table(data.guideProgress)
    local entry = type(guide.key) == "string" and progress and progress[guide.key]
    local step, stepId
    if Table(entry) then
        step, stepId = entry.step, entry.stepId
    elseif not entry and type(guide.name) == "string" and type(guide.group) == "string"
        and guide.name == data.currentGuideName and guide.group == data.currentGuideGroup then
        step, stepId = data.currentStep, data.currentStepId
    end
    step = PositiveInteger(step)
    local current = step and Table(guide.steps[step])
    if current and (stepId == nil or (PositiveInteger(stepId) and current.stepId == stepId)) then return step end
end

local function Delivered(id)
    local checker = C_QuestLog and C_QuestLog.IsQuestFlaggedCompleted or IsQuestFlaggedCompleted
    if type(checker) ~= "function" then return end
    local ok, value = pcall(checker, id)
    if ok and type(value) == "boolean" then return value end
end

local function Percent(value)
    return type(value) == "number" and value >= 0 and value <= 100
end

local function ElementLine(element, includePosition)
    if not Table(element) then return nil, true end
    local questAction = questActions[element.tag]
    local action = questAction or travelActions[element.tag]
    if not action then return nil, true end
    local id = PositiveInteger(element.questId)
    local delivered
    if questAction and id then delivered = Delivered(id) end
    if delivered == true then return end

    local detail = { action }
    if id then detail[#detail + 1] = "quest #" .. id end
    local objective = PositiveInteger(element.obj)
    if objective then detail[#detail + 1] = "objective " .. objective end
    if element.skip == true then
        detail[#detail + 1] = "skipped by RestedXP"
    elseif element.completed == true then
        detail[#detail + 1] = "marked complete by RestedXP"
    end
    if includePosition and Percent(element.x) and Percent(element.y) then
        local map = PositiveInteger(element.mapID)
        local zone = PositiveInteger(element.zone) or Text(element.zone)
        local location = map and ("mapID " .. map) or zone and ("RXP zone " .. zone)
        if location then detail[#detail + 1] = ("%s at %.1f, %.1f"):format(location, element.x, element.y) end
    end
    local text, clipped = Text(element.text)
    if not text then text, clipped = Text(element.rawtext) end
    if text then detail[#detail + 1] = 'guide text "' .. text .. '"' end
    return "  - " .. table.concat(detail, " | "), clipped or (questAction and (not id or delivered == nil))
end

function AzerMind.RestedXPRouteContext(options)
    local guide = Table(_G.RXP) and Table(_G.RXP.currentGuide)
    if not guide or guide.empty or not Table(guide.steps) then
        return "RestedXP route unavailable: no active RestedXP guide."
    end
    local data = Table(_G.RXPCData)
    local progress = data and Table(data.guideProgress)
    local player = Table(_G.RXP.player)
    if progress and player and type(progress.playerGUID) == "string" and type(player.guid) == "string"
        and progress.playerGUID ~= player.guid then
        return "RestedXP route unavailable: saved progress belongs to a different character."
    end
    local current = CurrentStep(guide, data)
    if not current then return "RestedXP route unavailable: current progress could not be matched to the active guide." end

    local name, nameClipped = Text(guide.name)
    local group, groupClipped = Text(guide.group)
    local version, versionClipped = Text(type(guide.version) == "number" and tostring(guide.version) or guide.version)
    local header = "RestedXP route: " .. (name or "unnamed guide") .. " | group " .. (group or "unknown")
        .. " | version " .. (version or "unknown") .. " | current step " .. current
    local lines = { header }
    local bytes = #header
    local partial = nameClipped or groupClipped or versionClipped
    local includePosition = not Table(options) or options.position ~= false
    local skips = Table(data.stepSkip) or {}
    local function Append(line)
        if bytes + #line + 1 > MAX_BYTES - #PARTIAL - 1 then
            partial = true
            return false
        end
        lines[#lines + 1] = line
        bytes = bytes + #line + 1
        return true
    end

    for index = current, math.min(current + 3, #guide.steps) do
        local step = Table(guide.steps[index])
        local status = index == current and "current" or "conditional lookahead"
        if not step then
            partial = true
        elseif skips[index] == true then
            if not Append("Step " .. index .. " (" .. status .. "): skipped by RestedXP") then break end
        else
            if step.completed == true then status = status .. ", marked complete by RestedXP" end
            if not Append("Step " .. index .. " (" .. status .. ")") then break end
            local elements = Table(step.elements) or {}
            local shown = false
            if #elements > MAX_ELEMENTS_PER_STEP then partial = true end
            for i = 1, math.min(#elements, MAX_ELEMENTS_PER_STEP) do
                local line, incomplete = ElementLine(elements[i], includePosition)
                partial = partial or incomplete
                if line then
                    if not Append(line) then break end
                    shown = true
                end
            end
            if not shown then Append("  No open action shown within this snapshot.") end
        end
    end
    if partial then lines[#lines + 1] = PARTIAL end
    return table.concat(lines, "\n")
end

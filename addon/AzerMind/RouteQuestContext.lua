-- Read live quest progress and catalog endpoints without changing tracking or the map.
local MAX_BYTES = 3500
local MAX_ACTIVE = 40
local MAX_READY = 8
local MAX_IN_PROGRESS = 4
local MAX_FINISHERS = 3
local MAX_POSITIONS = 3
local MAX_FOLLOW_UPS = 3

local function Read(fn, ...)
    if type(fn) ~= "function" then return nil end
    local ok, value, other = pcall(fn, ...)
    if ok then return value, other end
end

local function Table(value)
    return type(value) == "table" and value or {}
end

local function Number(value)
    return type(value) == "number" and value >= 0 and value < math.huge
end

local function ID(value)
    return Number(value) and value > 0 and value == math.floor(value)
end

local function Flag(value)
    if value == true or value == 1 then return true end
    if value == false or value == 0 then return false end
end

-- The history API is a fallback, read at most once per snapshot.
local function DeliveredLookup()
    local api = Table(C_QuestLog)
    local reported, completeHistory
    return function(id)
        local delivered = Flag(Read(api.IsQuestFlaggedCompleted or IsQuestFlaggedCompleted, id))
        if delivered ~= nil then return delivered end
        if not reported then
            reported = {}
            local ids = Read(api.GetAllCompletedQuestIDs)
            completeHistory = type(ids) == "table"
            for _, entry in pairs(Table(ids)) do
                if ID(entry) then reported[entry] = true else completeHistory = false end
            end
        end
        if reported[id] then return true end
        if completeHistory then return false end
    end
end

local function Positions(record, limit, areaLabel, partial)
    local spawns = Table(record.spawns)
    local areas, points = {}, {}
    for area in pairs(spawns) do
        if ID(area) then areas[#areas + 1] = area else partial() end
    end
    table.sort(areas)
    for _, area in ipairs(areas) do
        local positions = Table(spawns[area])
        for i = 1, math.min(#positions, MAX_POSITIONS) do
            local point = Table(positions[i])
            local x, y = point[1], point[2]
            if Number(x) and Number(y) and x <= 100 and y <= 100 then
                if #points == limit then partial(); return points end
                points[#points + 1] = areaLabel(area) .. string.format(" %.2f, %.2f", x, y)
            else
                partial()
            end
        end
        if #positions > MAX_POSITIONS then partial() end
    end
    return points
end

local function Finishers(record, world, items, add, label, areaLabel, partial)
    local groups = Table(record.finishedBy)
    local sources = {world.npcs, world.objects, items}
    local kinds = {"NPC", "object", "item"}
    local shown, mapped = 0, 0
    for group, kind in ipairs(kinds) do
        for _, id in ipairs(Table(groups[group])) do
            if shown == MAX_FINISHERS then partial(); return end
            if ID(id) then
                local entity = Table(Table(sources[group])[id])
                local name = label(entity.name, "name unavailable")
                local points = Positions(entity, MAX_POSITIONS - mapped, areaLabel, partial)
                mapped = mapped + #points
                shown = shown + 1
                if #points == 0 then partial() end
                local location = #points > 0 and table.concat(points, " / ") or "coordinates unavailable"
                add("  Catalog finisher " .. kind .. " #" .. id .. " " .. name .. " — " .. location)
            else
                partial()
            end
        end
    end
    if shown == 0 then add("  Finisher details unavailable in catalog"); partial() end
end

local function Objectives(objectives, add, label, partial)
    if type(objectives) ~= "table" then
        add("  Current objectives unavailable")
        partial()
        return
    end
    local shown = 0
    for i = 1, math.min(#objectives, 12) do
        local objective = Table(objectives[i])
        local finished = Flag(objective.finished)
        local count, total = objective.numFulfilled, objective.numRequired
        local hasCounts = Number(count) and Number(total) and total > 0
        if finished == nil and hasCounts then finished = count >= total end
        if not finished then
            if shown == 3 then partial(); break end
            shown = shown + 1
            local name = label(objective.text, "Unnamed objective")
            if hasCounts then
                local progress = count .. "/" .. total
                if not name:find(progress, 1, true) then name = name .. " (" .. progress .. ")" end
            end
            if finished == nil then name = name .. " (progress unknown)"; partial() end
            add("  Objective " .. name)
        end
    end
    if #objectives > 12 then partial() end
end

local function FollowUp(id, previous, details, index, add, label, partial)
    local record = Table(details[id] or index[id])
    local name = label(record.name, "name unavailable")
    local fields = {"after #" .. previous, "availability unverified"}
    if ID(record.requiredLevel) then
        fields[#fields + 1] = "minimum level " .. record.requiredLevel
    else
        fields[#fields + 1] = "minimum level unavailable"
        partial()
    end
    for _, field in ipairs({"preQuestSingle", "preQuestGroup"}) do
        local prereqs = Table(record[field])
        local ids = {}
        for i = 1, math.min(#prereqs, 4) do
            local prereq = prereqs[i]
            if type(prereq) == "number" and ID(math.abs(prereq)) then
                ids[#ids + 1] = "#" .. prereq
            else
                partial()
            end
        end
        if #prereqs > 4 then partial() end
        if #ids > 0 then fields[#fields + 1] = field .. ": " .. table.concat(ids, ", ") end
    end
    add("Conditional chain follow-up #" .. id .. " " .. name .. " (" .. table.concat(fields, " | ") .. ")")
end

function AzerMind.RouteQuestContext()
    local header = "Route quest evidence (current log + bundled catalog)"
    local footer = "Route quest evidence partial (some data unavailable or omitted)."
    local lines, bytes = {header}, #header
    local incomplete, full = false, false
    local function Partial() incomplete = true end
    local function Add(line)
        -- Keep related lines together in order even when the snapshot reaches its limit.
        if full then return end
        if bytes + #line + 1 + #footer + 1 > MAX_BYTES then full = true; Partial(); return end
        lines[#lines + 1] = line
        bytes = bytes + #line + 1
    end
    local function Label(value, fallback)
        if type(value) == "string" then
            local text = Read(AzerMind.PlainText, value) or value
            text = text:gsub("%c", " "):match("^%s*(.-)%s*$")
            if text ~= "" then
                if #text <= 112 then return text end
                local last = 109
                -- WoW Lua strings count bytes. Back up if the cut falls inside a UTF-8 character.
                local nextByte = text:byte(last + 1)
                while last > 0 and nextByte >= 128 and nextByte < 192 do
                    last = last - 1
                    nextByte = text:byte(last + 1)
                end
                Partial()
                return text:sub(1, last) .. "..."
            end
        end
        Partial()
        return fallback
    end
    local areaLabels = {}
    local function AreaLabel(area)
        if not areaLabels[area] then
            local name = Label(Read(AzerMind.QuestAreaName, area), "area name unavailable")
            local map = Read(AzerMind.QuestAreaMap, area)
            if not ID(map) then Partial() end
            areaLabels[area] = name .. " (area " .. area .. ", UI map " .. (ID(map) and map or "unavailable") .. ")"
        end
        return areaLabels[area]
    end

    local quests, partialLog = Read(AzerMind.ActiveQuests, {limit=MAX_ACTIVE, preserveUnknown=true})
    local ready, active, seen = {}, {}, {}
    local delivered = DeliveredLookup()
    local unknownDelivery = false
    if type(quests) ~= "table" or (partialLog and #quests == 0) then
        Add("Current active quest log unavailable.")
        Partial()
    else
        if partialLog or #quests > MAX_ACTIVE then Partial() end
        for i = 1, math.min(#quests, MAX_ACTIVE) do
            local q = Table(quests[i])
            if ID(q.id) and not seen[q.id] then
                seen[q.id] = true
                local isDelivered = delivered(q.id)
                if isDelivered == nil then unknownDelivery = true; Partial() end
                if isDelivered == false and Flag(q.complete) == true then
                    ready[#ready + 1] = q
                elseif isDelivered ~= true then
                    active[#active + 1] = q
                end
            elseif not ID(q.id) then
                Partial()
            end
        end
        if #ready + #active == 0 then Add("No undelivered active quests in the current log snapshot.") end
    end
    if unknownDelivery then Add("Some quests have delivered status unavailable from the client.") end
    if #ready > MAX_READY or #active > MAX_IN_PROGRESS then Partial() end

    local catalog = Table(#ready > 0 and Read(AzerMind.LoadCatalog, "QuestDetails"))
    local details = Table(catalog.quests)
    local world
    local items = Table(catalog.itemIndex or catalog.items)
    local index = Table(catalog.questIndex)
    local followUps = {}
    for i = 1, math.min(#ready, MAX_READY) do
        if full then break end
        local q = ready[i]
        local record = details[q.id]
        Add("Ready to turn in #" .. q.id .. " " .. Label(q.title, "title unavailable"))
        if type(record) == "table" then
            world = world or Table(Read(AzerMind.LoadCatalog, "World"))
            Finishers(record, world, items, Add, Label, AreaLabel, Partial)
            local nextId = record.nextQuestInChain
            if ID(nextId) and not seen[nextId] then
                if #followUps < MAX_FOLLOW_UPS then
                    local nextDelivered = delivered(nextId)
                    if nextDelivered == false then
                        seen[nextId] = true
                        followUps[#followUps + 1] = {id=nextId, previous=q.id}
                    elseif nextDelivered == nil then Partial() end
                else
                    Partial()
                end
            end
        else
            Add("  Quest finisher catalog unavailable for this quest.")
            Partial()
        end
    end
    for i = 1, math.min(#active, MAX_IN_PROGRESS) do
        if full then break end
        local q = active[i]
        local complete = Flag(q.complete)
        local status = ""
        if complete == true then
            status = " (objectives complete, delivered status unavailable)"
        elseif complete == nil then
            status = " (turn-in readiness unknown)"
            Partial()
        end
        Add("Active #" .. q.id .. " " .. Label(q.title, "title unavailable") .. status)
        Objectives(q.objectives, Add, Label, Partial)
    end
    for _, follow in ipairs(followUps) do
        FollowUp(follow.id, follow.previous, details, index, Add, Label, Partial)
    end
    if incomplete then lines[#lines + 1] = footer end
    return table.concat(lines, "\n")
end

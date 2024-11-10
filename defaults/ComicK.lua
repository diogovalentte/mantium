------------------------------
-- @name    ComicK
-- @url     https://comick.fun
-- @author  diogovalentte
-- @license MIT
------------------------------

----- IMPORTS -----
Http = require("http")
Json = require("json")
--- END IMPORTS ---

----- VARIABLES -----
Debug = false
Client = Http.client({ timeout = 20, insecure_ssl = true, debug = Debug })
ApiBase = "https://api.comick.fun"
ImageBase = "https://meo.comick.pictures"
Limit = 50
Lang = "en" -- Language: en = english, fr = french, etc.
Order = 0   -- Chapter Order: 0 = descending, 1 = ascending
--- END VARIABLES ---

----- MAIN -----

--- Searches for manga with given query.
-- @param query Query to search for
-- @return Table of tables with the following fields: name, url
function SearchManga(query)
    query = query:gsub(" ", "+")
    local request_url = ApiBase .. "/v1.0/search/?tachiyomi=true&q=" .. query
    local result = request(request_url)
    local result_body = Json.decode(result)

    local mangas = {}
    local i = 1

    for _, val in pairs(result_body) do
        local title = val["title"]

        if title ~= nil then
            local hid = val["hid"]
            local link = ApiBase .. "/comic/" .. tostring(hid) .. "/chapters"
            local manga = { url = link, name = title }

            mangas[i] = manga
            i = i + 1
        end
    end

    return mangas
end

--- Gets the list of all manga chapters.
-- @param mangaURL URL of the manga
-- @return Table of tables with the following fields: name, url

function MangaChapters(mangaURL)
    local request_url = mangaURL .. "?tachiyomi=true&lang=" .. Lang .. "&limit=" .. Limit .. "&chap-order=" .. Order
    local chapters = {}

    local i = 1

    local result = request(request_url)
    local result_body = Json.decode(result)
    local num_chapters = result_body["total"]
    local num_pages = math.ceil(num_chapters / Limit)

    local occurence = {}
    for j = 1, num_pages do
        local result = request(request_url .. "&page=" .. j)
        result_body = Json.decode(result)

        for _, val in pairs(result_body["chapters"]) do
            local hid = val["hid"]
            local num = val["chap"]
            if num == nil then
                num = 0
            end
            if not occurence[num] then
                occurence[num] = true

                local volume = tostring(val["vol"])
                if volume ~= "nil" then
                    volume = "Vol." .. volume
                else
                    volume = ""
                end
                local title = val["title"]
                local chap = "Chapter " .. tostring(num)
                local group_name = val["group_name"]

                if title then
                    chap = chap .. ": " .. tostring(title)
                end

                if group_name then
                    chap = chap .. " ["
                    for key, group in pairs(group_name) do
                        if key ~= 1 then
                            chap = chap .. ", "
                        end

                        chap = chap .. tostring(group)
                    end
                    chap = chap .. "]"
                end

                local link = ApiBase .. "/chapter/" .. tostring(hid)
                local chapter = { url = link, name = chap, volume = volume }

                chapters[i] = chapter
                i = i + 1
            end
        end
    end

    local reserved = {}
    local itemCount = #chapters
    for k = itemCount, 1, -1 do
        table.insert(reserved, chapters[k])
    end

    return reserved
end

--- Gets the list of all pages of a chapter.
-- @param chapterURL URL of the chapter
-- @return Table of tables with the following fields: url, index
function ChapterPages(chapterURL)
    local result = request(chapterURL)
    local result_body = Json.decode(result)
    local chapter_table = result_body["chapter"]

    local pages = {}
    local i = 1

    for key, val in pairs(chapter_table["md_images"]) do
        local ind = key
        local link = ImageBase .. "/" .. val["b2key"]
        local page = { url = link, index = ind }

        pages[i] = page
        i = i + 1
    end

    return pages
end

--- END MAIN ---

----- HELPERS -----
function request(url)
    local command = string.format('curl -s -H "User-Agent: %s" "%s"', UserAgent, url)
    local handle = io.popen(command)
    local response = handle:read("*a")
    handle:close()

    return response
end

--- END HELPERS ---

-- ex: ts=4 sw=4 et filetype=lua

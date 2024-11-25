--------------------------------------
-- @name    RawKuma
-- @url     https://rawkuma.com
-- @author  diogovalentte
-- @license MIT
--------------------------------------

----- IMPORTS -----
Html = require("html")
Http = require("http")
Json = require("json")
--- END IMPORTS ---

----- VARIABLES -----
Debug = false
Client = Http.client({ timeout = 20, insecure_ssl = true, debug = Debug })
Base = "https://rawkuma.com"
--- END VARIABLES ---

----- MAIN -----

--- Searches for manga with given query.
-- @param query Query to search for
-- @return Table of tables with the following fields: name, url
function SearchManga(query)
    query = query:gsub(" ", "+")
    local req_url = Base .. "/?s=" .. query
    local request = Http.request("GET", req_url)
    local result = Client:do_request(request)
    local doc = Html.parse(result.body)

    local mangas = {}

    doc:find("div.listupd > div > div"):each(function(i, el)
        local name = trim(el:find("a > div.bigor > div.tt"):text())
        local url = trim(el:find("a"):attr("href"))
        local manga = { url = url, name = name }
        mangas[i + 1] = manga
    end)

    return mangas
end

--- Gets the list of all manga chapters.
-- @param mangaURL URL of the manga
-- @return Table of tables with the following fields: name, url

function MangaChapters(mangaURL)
    local request = Http.request("GET", mangaURL)
    local result = Client:do_request(request)
    local doc = Html.parse(result.body)

    local chapters = {}

    doc:find("ul.clstyle > li"):each(function(i, el)
        local name = trim(el:find("div > div > a > span.chapternum"):text())
        local url = el:find("div > div > a"):attr("href")
        local chapter = { url = url, name = name }

        chapters[i + 1] = chapter
    end)

    reverseTableInPlace(chapters)

    return chapters
end

--- Gets the list of all pages of a chapter.
-- @param chapterURL URL of the chapter
-- @return Table of tables with the following fields: url, index
function ChapterPages(chapterURL)
    local request = Http.request("GET", chapterURL)
    local result = Client:do_request(request)
    local doc = Html.parse(result.body)

    local pages = {}

    doc:find("script"):each(function(_, el)
        local js = el:text()

        if not hasPrefix(js, "ts_reader.run(") then
            return nil
        end

        local json_text = trimPrefix(js, "ts_reader.run(")
        json_text = trimSuffix(json_text, ");")
        local json = Json.decode(json_text)

        local images = json["sources"][1]["images"]
        for i, image in ipairs(images) do
            local page = { url = image, index = i }
            pages[i] = page
        end
    end)

    if #pages == 0 then
        error("could not extract pages")
    end

    return pages
end

--- END MAIN ---

----- HELPERS -----
function trim(s)
    return (s:gsub("^%s*(.-)%s*$", "%1"))
end

function trimPrefix(s, prefix)
    if s:sub(1, #prefix) == prefix then
        return s:sub(#prefix + 1)
    else
        return s
    end
end

function trimSuffix(s, suffix)
    if s:sub(- #suffix) == suffix then
        return s:sub(1, - #suffix - 1)
    else
        return s
    end
end

function hasPrefix(s, prefix)
    return s:sub(1, #prefix) == prefix
end

function reverseTableInPlace(tbl)
    local left, right = 1, #tbl
    while left < right do
        tbl[left], tbl[right] = tbl[right], tbl[left]
        left = left + 1
        right = right - 1
    end
end

--- END HELPERS ---

-- ex: ts=4 sw=4 et filetype=lua

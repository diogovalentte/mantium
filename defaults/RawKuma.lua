--------------------------------------
-- @name    RawKuma
-- @url     https://rawkuma.net
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
Base = "https://rawkuma.net"
--- END VARIABLES ---

----- MAIN -----

--- Searches for manga with given query.
-- @param query Query to search for
-- @return Table of tables with the following fields: name, url
function SearchManga(query)
    local req_url = Base .. "/wp-admin/admin-ajax.php?action=advanced_search"
    local request = Http.request("POST", req_url, "query=" .. query .. "&orderBy=popular&order=desc")
    request:header_set("Content-Type", "application/x-www-form-urlencoded")

    local result = Client:do_request(request)
    local doc = Html.parse(result.body)

    local mangas = {}

    doc:find("div > div > a > img"):each(function(i, el)
        el = el:parent():parent()
        local name = trim(el:find("div > div > div > div > a"):text())
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
    local body = result.body

    -- extract manga ID
    local mangaInternalID = body:match("wp%-admin/admin%-ajax%.php%?manga_id=(%d+)")
    if not mangaInternalID then
        error("manga ID not found in HTML response")
    end

    local chapters = {}

    request =
        Http.request("GET", Base .. "/wp-admin/admin-ajax.php?page=1&action=chapter_list&manga_id=" .. mangaInternalID)
    result = Client:do_request(request)
    local doc = Html.parse(result.body)

    doc:find("div#chapter-list > div > a"):each(function(i, el)
        local name = trim(el:find("span"):text())
        local url = el:attr("href")
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

    doc:find("section > img"):each(function(_, el)
        local url = el:attr("src")
        if not url or url == "" then
            error("could not extract page URL")
        end
        local page = { url = url, index = #pages + 1 }
        pages[#pages + 1] = page
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

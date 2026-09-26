--------------------------------------
-- @name    JManga
-- @url     https://jmanga.locker
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
Base = "https://jmanga.locker"
DiscoverDomain = true
CurrentDomain = nil
--- END VARIABLES ---

----- MAIN -----

--- Searches for manga with given query.
-- @param query Query to search for
-- @return Table of tables with the following fields: name, url
function SearchManga(query)
    query = query:gsub(" ", "+")
    local req_url = getCurrentDomain() .. "/?q=" .. encode_query_args(query)
    local request = Http.request("GET", req_url)
    local result = Client:do_request(request)
    local doc = Html.parse(result.body)

    local mangas = {}

    doc:find("div.manga_list-sbs div.item"):each(function(i, el)
        local name_el = el:find("h3.manga-name > a")
        local name = trim(name_el:text())
        local url = name_el:attr("href")
        local manga = { url = url, name = name }
        mangas[i + 1] = manga
    end)

    return mangas
end

--- Gets the list of all manga chapters.
-- @param mangaURL URL of the manga
-- @return Table of tables with the following fields: name, url

function MangaChapters(mangaURL)
    mangaURL = rebaseURL(mangaURL)
    local request = Http.request("GET", mangaURL)
    local result = Client:do_request(request)
    local doc = Html.parse(result.body)

    local chapters = {}

    doc:find("ul#ja-chaps > li"):each(function(i, el)
        local name = extractChapter(el:find("span.name > strong"):text())
        local url = el:find("a"):attr("href")
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
    chapterURL = rebaseURL(chapterURL)
    local request = Http.request("GET", chapterURL)
    local result = Client:do_request(request)
    local doc = Html.parse(result.body)
    local pages = {}

    local chapter_number = extractChapterFromURL(chapterURL)
    if chapter_number == "" then
        error("could not extract chapter number from URL")
    end

    local data_id = ""
    doc:find('ul.reading-list > li[data-number="' .. chapter_number .. '"]'):each(function(_, el)
        data_id = el:attr("data-id")
    end)

    if data_id == "" then
        error("could not extract data-id")
    end

    request = Http.request("GET", getCurrentDomain() .. "/json/chapter?mode=vertical&id=" .. data_id)
    result = Client:do_request(request)
    local json = Json.decode(result.body)
    doc = Html.parse(json["html"])

    local i = 1
    doc:find("img"):each(function(_, el)
        local url = el:attr("data-src")
        if url == "" then
            error("could not extract image URL")
        end
        local page = { url = url, index = i }
        pages[i] = page
        i = i + 1
    end)

    if #pages == 0 then
        error("could not extract pages")
    end

    return pages
end

--- END MAIN ---

----- HELPERS -----
function reverseTableInPlace(tbl)
    local left, right = 1, #tbl
    while left < right do
        tbl[left], tbl[right] = tbl[right], tbl[left]
        left = left + 1
        right = right - 1
    end
end

function extractChapter(s)
    local match = string.match(s, "第(.-)話")
    if match then
        return match
    end
    error("could not extract chapter")
end

function extractChapterFromURL(url)
    return url:match("chapter%-(.-)%-raw")
end

function trim(s)
    return (s:gsub("^%s*(.-)%s*$", "%1"))
end

function encode_query_args(str)
    if str == nil then
        return ""
    end
    str = tostring(str)
    str = string.gsub(str, "\n", "\r\n")
    str = string.gsub(str, "([^%w%-_%.%~])", function(c)
        return string.format("%%%02X", string.byte(c))
    end)
    return str
end

-- The site moves to a new domain every few weeks and redirects the old ones.
-- Base is only the starting point: the current domain is read from the
-- canonical link of the page Base ends up at, and the manga and chapter URLs
-- are moved to it, since the old domains don't always keep the path.
-- Set DiscoverDomain to false to always use Base as it is.
function getCurrentDomain()
    if CurrentDomain ~= nil then
        return CurrentDomain
    end
    CurrentDomain = Base
    if not DiscoverDomain then
        return CurrentDomain
    end

    local ok, result = pcall(function()
        return Client:do_request(Http.request("GET", Base))
    end)
    if not ok or result == nil or result.body == nil then
        return CurrentDomain
    end

    local href = ""
    Html.parse(result.body):find('link[rel="canonical"]'):each(function(_, el)
        href = el:attr("href") or ""
    end)
    local domain = string.match(href, "^(https?://[^/]+)")
    if domain ~= nil and string.find(domain, "jmanga", 1, true) ~= nil then
        CurrentDomain = domain
    end

    return CurrentDomain
end

function rebaseURL(url)
    local domain = getCurrentDomain()
    return (string.gsub(url, "^https?://[^/]+", function()
        return domain
    end, 1))
end

--- END HELPERS ---

-- ex: ts=4 sw=4 et filetype=lua

--------------------------------------
-- @name    JManga
-- @url     https://jmanga.is
-- @author  diogovalentte
-- @license MIT
--------------------------------------

----- IMPORTS -----
Html = require("html")
Http = require("http")
Headless = require("headless")
Json = require("json")
Time = require("time")
--- END IMPORTS ---

----- VARIABLES -----
Debug = false
Client = Http.client({ timeout = 20, insecure_ssl = true, debug = Debug })
Browser = Headless.browser()
Page = Browser:page()
Delay = 5
Base = "https://jmanga.is"
ChapterHasNoImagesDefaultImage = "https://i.imgur.com/jMy7evE.jpeg"
--- END VARIABLES ---

----- MAIN -----

--- Searches for manga with given query.
-- @param query Query to search for
-- @return Table of tables with the following fields: name, url
function SearchManga(query)
    query = query:gsub(" ", "+")
    local req_url = Base .. "/?q=" .. query
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
    Page:navigate(chapterURL)
    Time.sleep(Delay)

    local pages = {}
    local doc = Html.parse(Page:html())
    local i = 1
    doc:find("div#vertical-content > div.loader > img"):each(function(_, el)
        local url = el:attr("data-src")
        if url == "" then
            url = ChapterHasNoImagesDefaultImage
        end
        local page = { url = url, index = i }
        pages[i] = page
        i = i + 1
    end)

    if #pages == 0 then
        pages[1] = { url = ChapterHasNoImagesDefaultImage, index = 1 }
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
    return ""
end

function trim(s)
    return (s:gsub("^%s*(.-)%s*$", "%1"))
end

--- END HELPERS ---

-- ex: ts=4 sw=4 et filetype=lua

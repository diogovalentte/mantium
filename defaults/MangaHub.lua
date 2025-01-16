------------------------------
-- @name    MangaHub
-- @url     https://mangahub.io
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
BaseSiteURL = "https://mangahub.io"
ApiBase = "https://api.mghcdn.com/graphql"
ImageBase = "https://imgx.mghcdn.com"
UserAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:30.0) Gecko/20100101 Firefox/30.0"
Limit = 50
Order = 0 -- Chapter Order: 0 = descending, 1 = ascending
--- END VARIABLES ---

----- MAIN -----

--- Searches for manga with given query.
-- @param query Query to search for
-- @return Table of tables with the following fields: name, url
function SearchManga(query)
    local req_query = '{"query":"{search(x:m01,q:\\"'
        .. query
        .. '\\",mod:POPULAR,limit:'
        .. Limit
        .. ',offset:0,count:true){rows{title,slug,image,latestChapter,status},count}}"}'
    local result = request(req_query)
    local json = Json.decode(result)
    local results = json["data"]["search"]["rows"]

    local mangas = {}
    for i, m in ipairs(results) do
        local manga = { url = BaseSiteURL .. "/manga/" .. m["slug"], name = m["title"] }
        mangas[i] = manga
    end

    return mangas
end

--- Gets the list of all manga chapters.
-- @param mangaURL URL of the manga
-- @return Table of tables with the following fields: name, url

function MangaChapters(mangaURL)
    local slug = getMangaSlug(mangaURL)
    if slug == "" then
        return {}
    end
    local req_query = '{"query":"{manga(x:m01,slug:\\"' .. slug .. '\\"){chapters{number,title}}}"}'
    local result = request(req_query)
    local json = Json.decode(result)
    local results = json["data"]["manga"]["chapters"]

    local chapters = {}
    for i, c in ipairs(results) do
        local chapter = {
            url = BaseSiteURL .. "/chapter/" .. slug .. "/chapter-" .. c["number"],
            name = "Chapter " .. c["number"],
        }
        chapters[i] = chapter
    end

    return chapters
end

--- Gets the list of all pages of a chapter.
-- @param chapterURL URL of the chapter
-- @return Table of tables with the following fields: url, index
function ChapterPages(chapterURL)
    local slug = getMangaSlugFromChapterURL(chapterURL)
    if slug == "" then
        return {}
    end
    local chapter_num = getChapterNumber(chapterURL)
    local req_query = '{"query":"{chapter(x:m01,slug:\\"'
        .. slug
        .. '\\",number:'
        .. chapter_num
        .. '){pages,manga{mainSlug}}}"}'
    local result = request(req_query)
    local json = Json.decode(result)
    local chapter = json["data"]["chapter"]

    local images = Json.decode(chapter["pages"])
    local imagePrefix = images["p"]
    local pages = {}
    for i, image in ipairs(images["i"]) do
        local url = ImageBase .. "/" .. imagePrefix .. image
        local page = { url = url, index = i }
        pages[i] = page
    end

    return pages
end

--- END MAIN ---

----- HELPERS -----
function request(body)
    local command = string.format(
        'curl -s -XPOST --tlsv1.2 -H "Content-Type: application/json" -H "Accept: application/json" -H "Origin: %s" -H "User-Agent: %s" -H "x-mhub-access: %s" "%s"',
        BaseSiteURL,
        UserAgent,
        generateUUID(),
        ApiBase
    )
    command = command .. " --data '" .. body .. "'"
    local handle = io.popen(command)
    local response = handle:read("*a")
    handle:close()

    return response
end

function generateUUID()
    local template = "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx"
    return string.gsub(template, "[xy]", function(c)
        local v = (c == "x") and math.random(0, 15) or math.random(8, 11)
        return string.format("%x", v)
    end)
end

function getMangaSlug(mangaURL)
    return mangaURL:match("manga/([^/?]+)")
end

function getMangaSlugFromChapterURL(chapterURL)
    return chapterURL:match("chapter/([^/?]+)")
end

function getChapterNumber(chapterURL)
    return chapterURL:match("chapter%-([%d%.]+)")
end

--- END HELPERS ---

-- ex: ts=4 sw=4 et filetype=lua

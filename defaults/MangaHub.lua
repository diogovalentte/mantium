------------------------------
-- @name    MangaHub
-- @url     https://mangahub.io
-- @author  diogovalentte
-- @license MIT
------------------------------

function generateUUID()
    local template = "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx"
    return string.gsub(template, "[xy]", function(c)
        local v = (c == "x") and math.random(0, 15) or math.random(8, 11)
        return string.format("%x", v)
    end)
end

----- IMPORTS -----
Http = require("http")
Json = require("json")
--- END IMPORTS ---

----- VARIABLES -----
Debug = false
BaseSiteURL = "https://mangahub.io"
ApiBase = "https://api.mghcdn.com/graphql"
ImageBase = "https://imgx.mghcdn.com"
UserAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:30.0) Gecko/20100101 Firefox/30.0"
Headers = {
    ["Content-Type"] = "application/json",
    ["Accept"] = "application/json",
    ["Origin"] = BaseSiteURL,
    ["Referer"] = BaseSiteURL .. "/",
    ["User-Agent"] = UserAgent,
    ["x-mhub-access"] = generateUUID(),
}
Client = Http.client({ timeout = 20, insecure_ssl = true, debug = Debug, headers = Headers })
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
    req_query =
    '{"query":"{search(x:m01,q:\\"dandadan\\",limit:10){rows{id,title,slug,image,rank,latestChapter,createdDate}}}"}'
    local request = Http.request("POST", ApiBase, req_query)
    local result, err = Client:do_request(request)
    if err then
        error(err)
    end
    if not (result.code == 200) then
        error("code: " .. result.code .. " - " .. result.body)
    end
    local result_body = Json.decode(result.body)
    local results = result_body["data"]["search"]["rows"]

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
    local request = Http.request("POST", ApiBase, req_query)
    local result, err = Client:do_request(request)
    if err then
        error(err)
    end
    if not (result.code == 200) then
        error("code: " .. result.code .. " - " .. result.body)
    end
    local result_body = Json.decode(result.body)
    local results = result_body["data"]["manga"]["chapters"]

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
    local request = Http.request("POST", ApiBase, req_query)
    local result, err = Client:do_request(request)
    if err then
        error(err)
    end
    if not (result.code == 200) then
        error("code: " .. result.code .. " - " .. result.body)
    end
    local result_body = Json.decode(result.body)
    local chapter = result_body["data"]["chapter"]

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

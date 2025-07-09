--------------------------------------
-- @name    KLManga
-- @url     https://klmanga.fi
-- @author  diogovalentte
-- @license MIT
--------------------------------------

----- IMPORTS -----
Html = require("html")
Http = require("http")
Json = require("json")
HttpUtil = require("http_util")
--- END IMPORTS ---

----- VARIABLES -----
Debug = false
Client = Http.client({ timeout = 20, insecure_ssl = true, debug = Debug })
Base = "https://klmanga.bot"
--- END VARIABLES ---

----- MAIN -----

--- Searches for manga with given query.
-- @param query Query to search for
-- @return Table of tables with the following fields: name, url
function SearchManga(query)
    query = HttpUtil.query_escape(query)
    local req_url = Base .. "/?s=" .. query
    local request = Http.request("GET", req_url)
    local result = Client:do_request(request)
    local doc = Html.parse(result.body)

    local mangas = {}

    doc:find("div.row > div.col-sm-4 > div.entry"):each(function(i, el)
        local name = trim(el:find("h2 > a"):text())
        name = trimSuffix(name, " (RAW – Free)")
        name = trimSuffix(name, " (RAW - Free)")
        name = trimSuffix(name, " (Raw – Free)")
        name = trimSuffix(name, " (Raw - Free)")
        local url = trim(el:find("h2 > a"):attr("href"))
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

    doc:find("div.chapter-box > h4 > a"):each(function(i, el)
        local name = el:find("span"):text()
        local chapter_num = extractChapter(name)
        name = "Chapter " .. chapter_num
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

    local chapter_id_js = ""
    local nonce_a_js = ""
    doc:find("script"):each(function(_, el)
        if chapter_id_js ~= "" and nonce_a_js ~= "" then
            return
        end

        local js = el:text()
        if contains(js, "decode_images_g") then
            chapter_id_js = js
        elseif contains(js, [["nonce_a":"]]) then
            nonce_a_js = js
        end
    end)

    if chapter_id_js == "" or nonce_a_js == "" then
        error("could not find necessary data in the HTML")
    end

    local chapter_id = extractChapterID(chapter_id_js)
    local number = extractNumber(chapter_id_js)
    local nonce_a = extractNonceA(nonce_a_js)

    local current_domain = getCurrentDomain()

    local req_url = current_domain .. "/wp-admin/admin-ajax.php"
    local page_idx = 1
    local img_index = 0
    while true do
        local data = "action=z_do_ajax&_action=decode_images_g&chapter_id="
            .. chapter_id
            .. "&p="
            .. number
            .. "&img_index="
            .. img_index
            .. "&nonce_a="
            .. nonce_a

        request = Http.request("POST", req_url, data)
        request:header_set("content-type", "application/x-www-form-urlencoded; charset=UTF-8")
        result = Client:do_request(request)

        local json = Json.decode(result.body)
        local going = json["going"]
        img_index = json["img_index"]
        local image_urls = extractImageURLs(json["mes"])

        for _, url in ipairs(image_urls) do
            local page = { url = url, index = page_idx }
            table.insert(pages, page)
            page_idx = page_idx + 1
        end

        if going == 0 then
            break
        end
    end

    if #pages == 0 then
        error("no pages found")
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

function extractChapter(s)
    local match = string.match(s, "第(.-)話")
    if match then
        return match
    end
    error("could not extract chapter")
end

function extractChapterID(s)
    local chapter_id = string.match(s, "chapter_id:%s*'([^']+)'")
    if chapter_id then
        return chapter_id
    end
    error("could not find chapter id")
end

function extractNonceA(s)
    local nonce_a = string.match(s, [["nonce_a":"(.-)"]])
    if nonce_a then
        return nonce_a
    end
    error("could not find nonce_a")
end

function extractNumber(s)
    local number = string.match(s, "p:%s*([^,]+),")
    if number then
        return number
    end
    error("could not extract number")
end

function contains(str, substring)
    return string.find(str, substring) ~= nil
end

function extractImageURLs(str)
    local urls = {}
    for url in string.gmatch(str, "src=[\"'](https?://[^\"']+)[\"']") do
        table.insert(urls, url)
    end
    return urls
end

function reverseTableInPlace(tbl)
    local left, right = 1, #tbl
    while left < right do
        tbl[left], tbl[right] = tbl[right], tbl[left]
        left = left + 1
        right = right - 1
    end
end

-- Get the current domain of the website.
-- The domain varies a lot.
function getCurrentDomain()
    local request = Http.request("GET", Base)
    local result = Client:do_request(request)
    local doc = Html.parse(result.body)
    local domain = ""

    doc:find("header.site-header"):each(function(_, el)
        domain = el:find("img"):parent():attr("href")
    end)

    if domain == "" then
        error("could not find domain")
    end

    return domain
end

--- END HELPERS ---

-- ex: ts=4 sw=4 et filetype=lua

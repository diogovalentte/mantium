default_sources = {
    "Mangadex": "mangadex",
    "Mangaplus": "mangaplus",
    "Mangahub": "mangahub",
    "MangaUpdates": "mangaupdates",
    "RawKuma": "rawkuma",
    "KLManga": "klmanga",
    "JManga": "jmanga",
}
reversed_default_sources = {v: k for k, v in default_sources.items()}

manga_status_options = {
    0: "📚 All",
    1: "📖 Reading",
    2: "✅ Completed",
    3: "🚧 On Hold",
    4: "❌ Dropped",
    5: "📅 Plan to Read",
}
sort_options = [
    "Name",
    "Unread",
    "Last Read",
    "Chapters Released",
    "Released Chapter Date",
]
chapter_link_tag_background_color = "rgb(219, 233, 254)"
chapter_link_tag_text_color = "rgb(59, 130, 246)"

grid_view_rows_per_page = 3
list_view_rows_per_page = 20
display_modes = ("Grid View", "List View")
columns_min_value = 1
columns_max_value = 10
search_results_limit_min_value = 1
search_results_limit_max_value = 50

CUSTOM_MANGA_SOURCE = "custom_manga"
# Chapter of custom mangas tracked only by the chapter URL
UNKNOWN_CHAPTER = "?"
DEFAULT_MANGA_COVER = "https://i.imgur.com/jMy7evE.jpeg"

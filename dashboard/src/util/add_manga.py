from typing import Any
import hashlib
from urllib.parse import urlparse

import src.util.defaults as defaults
import streamlit as st
from src.api.api_client import get_api_client
from src.exceptions import APIException
from src.util.util import (
    centered_container,
    escape_html,
    get_api_error_detail,
    get_api_user_message,
    get_invalid_url_message,
    get_logger,
    get_relative_time,
    get_updated_at_datetime,
    safe_href,
    tagger,
    set_is_dialog_open,
)
from src.util.defaults import reversed_default_sources
from streamlit import session_state as ss


logger = get_logger()


def show_add_manga_form(form_type: str):
    if ss.get("add_manga_chapter_options", None) is not None:
        del ss["add_manga_chapter_options"]
    if ss.get("add_manga_search_selected_manga", None) is not None:
        del ss["add_manga_search_selected_manga"]
    if ss.get("add_manga_form_url", "") != "":
        ss["add_manga_form_url"] = ""
    if ss.get("add_manga_manga_to_add", None) is not None:
        del ss["add_manga_manga_to_add"]
    base_key = "add_manga_search_selected_manga_search_results"
    ss[base_key + "_mangadex"] = {}
    ss[base_key + "_mangaplus"] = {}
    ss[base_key + "_mangahub"] = {}
    ss[base_key + "_mangaupdates"] = {}
    ss[base_key + "_rawkuma"] = {}
    ss[base_key + "_klmanga"] = {}
    ss[base_key + "_jmanga"] = {}
    ss["add_manga_search_go_back_to_tab"] = None

    if form_type == "url":

        @st.dialog("Add Manga", on_dismiss=set_is_dialog_open)
        def show():
            ss["is_dialog_open"] = True
            e = st.empty()
            if ss.get("add_manga_manga_to_add", None) is not None:
                with e:
                    add_manga()
                    del ss["add_manga_manga_to_add"]
            else:
                with e.container():
                    show_add_manga_form_url()

    elif form_type == "search":

        @st.dialog("Add Manga", width="medium", on_dismiss=set_is_dialog_open)
        def show():
            ss["is_dialog_open"] = True
            e = st.empty()
            if ss.get("add_manga_manga_to_add", None) is not None:
                with e:
                    add_manga()
                    del ss["add_manga_manga_to_add"]
            else:
                with e.container():
                    show_add_manga_form_search()

    elif form_type == "custom":

        @st.dialog("Add Custom Manga", on_dismiss=set_is_dialog_open)
        def show():
            ss["is_dialog_open"] = True
            e = st.empty()
            with e.container():
                show_add_custom_manga_form(0)

    else:
        st.stop()

    show()


def add_manga():
    api_client = get_api_client()
    ex = None
    with st.spinner("Adding manga..."):
        try:
            api_client.add_multimanga(
                ss["add_manga_manga_to_add"]["name"],
                ss["add_manga_manga_to_add"]["url"],
                ss["add_manga_manga_to_add"]["status"],
                ss["add_manga_manga_to_add"]["manga_internal_id"],
                ss["add_manga_manga_to_add"]["cover_img_url"],
                ss["add_manga_manga_to_add"]["cover_img"],
                ss["add_manga_manga_to_add"]["last_read_chapter"]["chapter"],
                ss["add_manga_manga_to_add"]["last_read_chapter"]["url"],
                ss["add_manga_manga_to_add"]["last_read_chapter"]["internal_id"],
                ss["add_manga_manga_to_add"][
                    "last_released_chapter_name_selector"
                ]["selector"],
                ss["add_manga_manga_to_add"][
                    "last_released_chapter_name_selector"
                ]["attribute"],
                ss["add_manga_manga_to_add"][
                    "last_released_chapter_name_selector"
                ]["regex"],
                ss["add_manga_manga_to_add"][
                    "last_released_chapter_name_selector"
                ]["get_first"],
                ss["add_manga_manga_to_add"][
                    "last_released_chapter_url_selector"
                ]["selector"],
                ss["add_manga_manga_to_add"][
                    "last_released_chapter_url_selector"
                ]["attribute"],
                ss["add_manga_manga_to_add"][
                    "last_released_chapter_url_selector"
                ]["get_first"],
                ss["add_manga_manga_to_add"][
                    "last_released_chapter_use_browser"
                ],

            )
        except Exception as e:
            ex = e

    ss["add_manga_search_selected_manga"] = None
    ss["add_manga_form_url"] = ""
    if ex is not None:
        resp_text = str(ex).lower()
        logger.exception(ex)
        if (
            "multimang added to DB, but error executing integrations:".lower()
            in str(resp_text).lower()
        ):
            logger.warning(resp_text)
            ss[
                "add_manga_warning_message"
            ] = "Manga added to DB, but couldn't add it to at least one integration"
            st.rerun()
        elif "manga already exists in DB".lower() in str(resp_text).lower():
            st.warning("Manga already in Mantium")
        elif "source" in str(resp_text).lower() and "is not allowed" in str(resp_text).lower():
            st.warning("Not allowed to add mangas from this source")
        elif (
            "duplicate key value violates unique constraint" in resp_text
            and "chapters_pkey" in resp_text
        ):
            st.warning("Last read chapter URL already in Mantium")
        elif user_message := get_api_user_message(ex):
            st.warning(user_message)
        else:
            st.error(f"Error while adding manga: {get_api_error_detail(ex)}")
    else:
        ss["add_manga_success_message"] = "Manga added successfully"
        st.rerun()


def show_bottom_add_manga_form(manga_url: str, manga_internal_id: str):
    with st.form(key="add_manga_form", border=False, clear_on_submit=True):
        st.selectbox(
            "Status",
            index=0,
            options=list(defaults.manga_status_options.keys())[
                1:
            ],  # Exclude the "All" option
            format_func=lambda index: defaults.manga_status_options[index],
            key="add_manga_form_status",
        )

        st.selectbox(
            "Last Read Chapter",
            options=ss.get("add_manga_chapter_options", []),
            index=None,
            key="add_manga_form_chapter",
            format_func=lambda chapter: (
                f"Ch. {chapter['Chapter']}{(' (' + get_relative_time(get_updated_at_datetime(chapter['UpdatedAt']))) + ')' if chapter['UpdatedAt'] != '0001-01-01T00:00:00Z' else ''}"
                if chapter is not None
                else "N/A"
            ),
        )

        if (
            ss.get("add_manga_chapter_options") is not None
            and len(ss.get("add_manga_chapter_options", [])) < 1
        ):
            st.warning("Manga has no released chapters. You still can add it.")

        def add_manga_callback():
            chapter = ss.add_manga_form_chapter
            if chapter is None:
                chapter = {}
            ss["add_manga_manga_to_add"] = {
                "name": "",
                "url": manga_url,
                "status": ss.add_manga_form_status,
                "manga_internal_id": manga_internal_id,
                "cover_img_url": "",
                "cover_img": None,
                "last_read_chapter": {
                    "chapter": chapter.get("Chapter", ""),
                    "url": chapter.get("URL", ""),
                    "internal_id": chapter.get("InternalID", ""),
                },
                "last_released_chapter_name_selector": {
                    "selector": "",
                    "attribute": "",
                    "regex": "",
                    "get_first": False,
                },
                "last_released_chapter_url_selector": {
                    "selector": "",
                    "attribute": "",
                    "get_first": False,
                },
                "last_released_chapter_use_browser": False,
            }
            del ss["add_manga_chapter_options"]

        st.form_submit_button(
            "Add",
            on_click=add_manga_callback,
            use_container_width=True,
            type="primary",
        )


def show_add_manga_form_url():
    api_client = get_api_client()
    manga_url = st.text_input(
        "Manga URL",
        placeholder="https://mangahub.io/manga/one-piece",
        key="add_manga_form_url",
    )

    if manga_url:
        try:
            with st.spinner("Getting manga chapters..."):
                ss["add_manga_chapter_options"] = api_client.get_cached_manga_chapters(
                    -1, manga_url, ""
                )
        except APIException as e:
            resp_text = str(e.response_text).lower()
            if "source not found" in resp_text:
                st.warning("No source site for this manga")
            elif (
                "manga doesn't have and id or url" in resp_text
                or "invalid uri for request" in resp_text
            ):
                st.warning("Invalid URL")
            elif "manga not found in source" in resp_text:
                st.warning("Manga not found")
            elif "one of the manga attributes were not found in source" in resp_text:
                st.warning("Manga not found")
            else:
                logger.exception(e)
                st.error("Error while getting manga chapters.")
                st.stop()
        else:
            show_bottom_add_manga_form(ss.add_manga_form_url, "")


def show_add_manga_form_search():
    api_client = get_api_client()
    sources = {}
    for name, source in defaults.default_sources.items():
        if source in ss["configs"]["manga"]["allowedSources"]:
            sources[name] = defaults.default_sources[name]

    container = st.empty()
    if ss.get("add_manga_search_selected_manga", None) is not None:
        with container:
            try:
                with st.spinner("Getting manga chapters..."):
                    ss[
                        "add_manga_chapter_options"
                    ] = api_client.get_cached_manga_chapters(
                        -1,
                        ss["add_manga_search_selected_manga"]["URL"],
                        ss["add_manga_search_selected_manga"]["InternalID"],
                    )
            except APIException as e:
                logger.exception(e)
                st.error("Error while getting manga chapters.")
                st.stop()

            show_bottom_add_manga_form(
                ss["add_manga_search_selected_manga"]["URL"],
                ss["add_manga_search_selected_manga"]["InternalID"],
            )

        def on_click():
            ss["add_manga_search_go_back_to_tab"] = reversed_default_sources[ss["add_manga_search_selected_manga"]["Source"]]
            ss["add_manga_search_selected_manga"] = None

        st.button("Back", use_container_width=True, on_click=on_click)
    else:
        # if change key_to_save_manga, also change it in func show_dialogs in the 01_?.py main file
        button_name, key_to_save_manga = "Select", "add_manga_search_selected_manga"
        with container:
            tabs = st.tabs(list(sources.keys()), default=ss["add_manga_search_go_back_to_tab"])
            for i, source in enumerate(sources.keys()):
                with tabs[i]:
                    show_search_manga_term_form(
                        sources[source], button_name, key_to_save_manga
                    )


def show_search_manga_term_form(source: str, button_name: str, key_to_save_manga: str):
    """Show search manga term form.

    Args:
        source(str): The source name to search for manga.
        button_name (str): The name of the button to select a manga.
        key_to_save_manga (str): The key to save the selected manga in streamlit.session_state.
    """
    api_client = get_api_client()
    search_results_key = f"{key_to_save_manga}_search_results_{source}"
    search_term_key = f"{key_to_save_manga}_search_term_{source}"

    term = st.text_input(
        "Term to Search",
        value=(
            ss[search_term_key]
            if ss.get(search_term_key, "") != ""
            else ss.get(search_results_key, {}).get("term", "")
        ),
        key=search_term_key,
    )

    if term == "" or term is None:
        ss[search_results_key]["term"] = term
        return
    elif ss[search_results_key].get("term", "") == term:
        results = ss[search_results_key].get("results", [])
    else:
        try:
            with st.spinner("Searching..."):
                results = api_client.search_mangas(
                    term,
                    ss["configs"]["display"]["searchResultsLimit"],
                    source,
                )
                ss[search_results_key]["results"] = results
        except Exception as ex:
            logger.exception(ex)
            st.error("Error while searching for manga.")
            st.stop()
        else:
            ss[search_results_key]["term"] = term

    if len(results) == 0:
        st.warning("No results found.")
    else:
        columns_num = 2 if not ss["is_mobile"] else 1
        show_search_result_mangas(
            st.columns(columns_num), results, button_name, key_to_save_manga
        )
        st.info(
            "Did not find the manga you were looking for? Try another source site or using the URL directly."
        )


def show_search_result_mangas(
    cols_list: list, mangas, button_name: str, key_to_save_manga: str
):
    """Show search result mangas in the cols_list columns.

    Args:
        cols_list (list): A list of streamlit.columns.
        mangas (dict): A list of search result mangas.
        button_name (str): The name of the button to select a manga.
        key_to_save_manga (str): The key to save the selected manga in streamlit.session_state.
    """
    manga_container_height = 660
    col_index = 0
    # Some sources return the same manga more than once - JManga answers
    # "one" with OMEGA ONE twice, under two spellings and two cover URLs.
    # Showing it twice is wrong on its own, and two entries sharing a URL and
    # a cover would collide on the select button's key and raise
    # StreamlitDuplicateElementKey, taking the whole search down.
    seen_urls = set()
    for manga in mangas:
        if manga["URL"] in seen_urls:
            continue
        seen_urls.add(manga["URL"])

        if col_index == len(cols_list):
            col_index = 0
        with cols_list[col_index]:
            with st.container(border=True, height=manga_container_height):
                # Japanese characters are replaced with "-" in the key, so we need to hash the URL to get a unique key for each manga
                key = (
                    "center_container_show_search_result_mangas"
                    + hashlib.md5(manga["URL"].encode()).hexdigest()
                )
                with centered_container(key):
                    show_search_result_manga(manga, button_name, key_to_save_manga)
        col_index += 1


def show_search_result_manga(
    manga: dict[str, Any], button_name: str, key_to_save_manga: str
):
    """Show search result manga.

    Args:
        mangas (dict): A list of search result mangas.
        button_name (str): The name of the button to select a manga.
        key_to_save_manga (str): The key to save the selected manga in streamlit.session_state.
    """
    # Try to make the title fit in the container the best way
    # Also try to make the containers the same size
    default_title_font_size = 36
    title_len = len(manga["Name"])
    if title_len < 15:
        font_size = default_title_font_size
        margin = 0
    elif title_len < 30:
        font_size = 20
        margin = (default_title_font_size - font_size) / 2 + 1.6
    else:
        font_size = 15
        margin = (default_title_font_size - font_size) / 2 + 1.6
    improve_headers = """
        <style>
            /* Hide the header link button */
            h1.manga_header > div > a {
                display: none !important;
            }
            /* Add ellipsis (...) if the manga name is to long */
            h1.manga_header {
                white-space: nowrap !important;
                overflow: hidden !important;
                text-overflow: ellipsis !important;
            }

            h1.manga_header {
                padding: 0px 0px 1rem;
            }

            a.manga_header {
                text-decoration: none;
                color: inherit;
            }
            a.manga_header:hover {
                color: #04c9b7;
            }
        </style>
    """
    st.markdown(improve_headers, unsafe_allow_html=True)
    st.markdown(
        f"""<h1
            class="manga_header" style='text-align: center; margin-top: {margin}px; margin-bottom: {margin}px; font-size: {font_size}px;'>
                <a class="manga_header" href="{safe_href(manga["URL"])}" target="_blank">{escape_html(manga["Name"])}</a>
            </h1>
        """,
        unsafe_allow_html=True,
    )

    if manga["CoverURL"] != "":
        st.markdown(
            f"""<img src="{manga["CoverURL"]}" width="250" height="355"/>""",
            unsafe_allow_html=True,
        )
    else:
        st.markdown(
            f"""<img src="{defaults.DEFAULT_MANGA_COVER}" width="250" height="355"/>""",
            unsafe_allow_html=True,
        )
    # Hide the "View fullscreen" button from the image
    hide_img_fs = """
    <style>
        button[title="View fullscreen"]{
            display: none !important;
        }
    </style>
    """
    st.markdown(hide_img_fs, unsafe_allow_html=True)

    tag_content_format = f"""
        <span style="color: {defaults.chapter_link_tag_text_color}">{{}}</span>
    """

    status = tag_content_format.format(
        manga["Status"].capitalize() if manga["Status"] != "" else "N/A",
    )
    tagger(
        "<strong>Status:</strong>",
        status,
        defaults.chapter_link_tag_background_color,
        "float: right;",
    )

    year = tag_content_format.format(
        manga["Year"] if manga["Year"] not in ("", "0", 0) else "N/A",
    )
    tagger(
        "<strong>Year:</strong>",
        year,
        defaults.chapter_link_tag_background_color,
        "float: right;",
    )

    if manga["LastChapterURL"] == "":
        last_chapter = tag_content_format.format(
            escape_html(manga["LastChapter"]) if manga["LastChapter"] not in ("", "0") else "N/A",
        )
    else:
        last_chapter = f"""
            <a href="{safe_href(manga["LastChapterURL"])}" target="_blank" style="text-decoration: none; color: {defaults.chapter_link_tag_text_color}">
                <span>{escape_html(manga["LastChapter"]) if manga["LastChapter"] not in ("", "0") else "N/A"}</span>
            </a>
        """
    tagger(
        "<strong>Last Chapter:</strong>",
        last_chapter,
        defaults.chapter_link_tag_background_color,
        "float: right;",
    )

    st.caption(manga["Description"])

    def on_click():
        ss[key_to_save_manga] = manga

    st.button(
        button_name,
        type="primary",
        use_container_width=True,
        on_click=on_click,
        key=key_to_save_manga + "_search_result_" + manga["URL"] + manga["CoverURL"],
    )


def test_custom_manga_selectors(
    manga_url: str,
    name_selector: str,
    name_attribute: str,
    name_regex: str,
    name_get_first: bool,
    url_selector: str,
    url_attribute: str,
    url_get_first: bool,
    use_browser: bool,
):
    """Shows the last released chapter the selectors get from the manga page, without saving anything."""
    if manga_url == "":
        st.warning("Provide the manga URL to test the selectors")
        return
    if invalid_url_message := get_invalid_url_message("manga URL", manga_url):
        st.warning(invalid_url_message)
        return
    if name_selector == "" and url_selector == "":
        st.warning("Provide at least one selector to test")
        return

    try:
        with st.spinner("Testing selectors..."):
            chapter = get_api_client().test_custom_manga_selectors(
                manga_url,
                name_selector,
                name_attribute,
                name_regex,
                name_get_first,
                url_selector,
                url_attribute,
                url_get_first,
                use_browser,
            )
    except Exception as ex:
        if user_message := get_api_user_message(ex):
            st.warning(user_message)
        else:
            logger.exception(ex)
            st.error(f"Error while testing the selectors: {get_api_error_detail(ex)}")
        return

    st.success("The selectors work. Last released chapter found:")
    lines = []
    if name_selector != "":
        lines.append(f"Chapter: {chapter['Chapter']}")
    if url_selector != "":
        lines.append(f"URL: {chapter['URL']}")
    st.text("\n".join(lines))


# Session state keys (after the form key prefix) of the custom manga form fields that a preset fills.
custom_manga_preset_fields = {
    "last_released_chapter_name_selector": ("name_selector", "Selector", ""),
    "last_released_chapter_name_attribute": ("name_selector", "Attribute", ""),
    "last_released_chapter_name_regex": ("name_selector", "Regex", ""),
    "last_released_chapter_name_get_first": ("name_selector", "GetFirst", False),
    "last_released_chapter_url_selector": ("url_selector", "Selector", ""),
    "last_released_chapter_url_attribute": ("url_selector", "Attribute", ""),
    "last_released_chapter_url_get_first": ("url_selector", "GetFirst", False),
}


def get_custom_manga_presets(key_prefix: str, reload: bool = False) -> dict[int, dict[str, Any]]:
    """Returns the custom manga presets by ID, loaded once per form."""
    if reload or key_prefix + "presets" not in ss:
        presets = get_api_client().get_custom_manga_presets()
        ss[key_prefix + "presets"] = {preset["id"]: preset for preset in presets}
    return ss[key_prefix + "presets"]


def custom_manga_selector_fields_are_empty(key_prefix: str) -> bool:
    return all(
        not ss.get(key_prefix + field, "")
        for field in ("last_released_chapter_name_selector", "last_released_chapter_url_selector")
    )


def apply_custom_manga_preset(key_prefix: str, preset_id: int | None = None):
    """Fills the selector fields with the preset. Used as a widget callback."""
    if preset_id is None:
        preset_id = ss[key_prefix + "preset"]
    else:
        ss[key_prefix + "preset"] = preset_id
    if preset_id is None:
        return

    preset = get_custom_manga_presets(key_prefix)[preset_id]
    for field, (selector_key, attribute, default) in custom_manga_preset_fields.items():
        selector = preset[selector_key] or {}
        ss[key_prefix + field] = selector.get(attribute, default)
    ss[key_prefix + "last_released_chapter_use_browser"] = preset["use_browser"]


def on_custom_manga_url_change(key_prefix: str):
    ss[key_prefix + "detect_presets"] = True
    ss[key_prefix + "preset_detections"] = None
    ss[key_prefix + "new_preset_domains"] = urlparse(ss.get(key_prefix + "url", "")).hostname or ""


def detect_custom_manga_presets(key_prefix: str):
    """Tests the presets against the manga URL, and selects one if the selector fields are empty."""
    manga_url = ss.get(key_prefix + "url", "")
    if manga_url == "" or get_invalid_url_message("manga URL", manga_url):
        return

    try:
        with st.spinner("Looking for a preset that works for this site..."):
            result = get_api_client().detect_custom_manga_presets(manga_url)
    except Exception as ex:
        ss[key_prefix + "preset_detections"] = {
            "error": get_api_user_message(ex) or get_api_error_detail(ex),
        }
        if get_api_user_message(ex) is None:
            logger.exception(ex)
        return

    working = [d for d in result["detections"] if d["chapter"] is not None]
    domain_presets = result["domain_presets"]
    ss[key_prefix + "preset_detections"] = {"working": working, "domain_presets": domain_presets}

    # A preset saved for this site wins over the ones that only happen to work
    chosen = None
    if domain_presets:
        chosen = domain_presets[0]["id"]
    elif working:
        chosen = working[0]["preset"]["id"]
    if chosen is not None and custom_manga_selector_fields_are_empty(key_prefix):
        get_custom_manga_presets(key_prefix, reload=True)
        apply_custom_manga_preset(key_prefix, chosen)
        ss[key_prefix + "preset_auto_selected"] = True


def save_custom_manga_preset(key_prefix: str):
    name_selector = ss.get(key_prefix + "last_released_chapter_name_selector", "")
    url_selector = ss.get(key_prefix + "last_released_chapter_url_selector", "")
    preset: dict[str, Any] = {
        "name": ss.get(key_prefix + "new_preset_name", ""),
        "domains": [d for d in ss.get(key_prefix + "new_preset_domains", "").split(",") if d.strip()],
        "use_browser": ss.get(key_prefix + "last_released_chapter_use_browser", False),
    }
    if name_selector:
        preset["name_selector"] = {
            "selector": name_selector,
            "attribute": ss.get(key_prefix + "last_released_chapter_name_attribute", ""),
            "regex": ss.get(key_prefix + "last_released_chapter_name_regex", ""),
            "get_first": ss.get(key_prefix + "last_released_chapter_name_get_first", False),
        }
    if url_selector:
        preset["url_selector"] = {
            "selector": url_selector,
            "attribute": ss.get(key_prefix + "last_released_chapter_url_attribute", ""),
            "get_first": ss.get(key_prefix + "last_released_chapter_url_get_first", False),
        }

    try:
        saved = get_api_client().add_custom_manga_preset(preset)
    except Exception as ex:
        if (user_message := get_api_user_message(ex)) is None:
            logger.exception(ex)
        ss[key_prefix + "preset_message"] = ("warning", user_message or f"Error while saving the preset: {get_api_error_detail(ex)}")
        return

    get_custom_manga_presets(key_prefix, reload=True)
    ss[key_prefix + "preset"] = saved["id"]
    ss[key_prefix + "preset_message"] = ("success", f'Preset "{saved["name"]}" saved')


def delete_custom_manga_preset(key_prefix: str):
    preset_id = ss.get(key_prefix + "preset")
    try:
        get_api_client().delete_custom_manga_preset(preset_id)
    except Exception as ex:
        if (user_message := get_api_user_message(ex)) is None:
            logger.exception(ex)
        ss[key_prefix + "preset_message"] = ("warning", user_message or f"Error while deleting the preset: {get_api_error_detail(ex)}")
        return

    get_custom_manga_presets(key_prefix, reload=True)
    ss[key_prefix + "preset"] = None
    ss[key_prefix + "preset_message"] = ("success", "Preset deleted")


def show_custom_manga_presets(key_prefix: str):
    """Shows the preset picker of the custom manga form.

    Picking a preset fills the last released chapter selectors. When the manga URL
    changes, the presets are tested against the page, and the one that works is
    picked if the selector fields are still empty.
    """
    try:
        presets = get_custom_manga_presets(key_prefix)
    except Exception as ex:
        logger.exception(ex)
        st.warning(f"Couldn't load the presets: {get_api_error_detail(ex)}")
        return

    if ss.pop(key_prefix + "detect_presets", False):
        detect_custom_manga_presets(key_prefix)
        presets = get_custom_manga_presets(key_prefix)

    if ss.get(key_prefix + "preset") not in presets:
        ss[key_prefix + "preset"] = None

    preset_col, detect_col = st.columns([3, 1], vertical_alignment="bottom")
    with preset_col:
        st.selectbox(
            "Preset",
            options=[None] + list(presets.keys()),
            format_func=lambda preset_id: "None" if preset_id is None else presets[preset_id]["name"],
            help="Fills the last released chapter selectors with ones that work for many sites. You can still edit them.",
            key=key_prefix + "preset",
            on_change=apply_custom_manga_preset,
            args=(key_prefix,),
        )
    with detect_col:
        st.button(
            "Detect",
            key=key_prefix + "detect_presets_button",
            use_container_width=True,
            help="Test every preset against the manga URL.",
            on_click=on_custom_manga_url_change,
            args=(key_prefix,),
        )

    selected = presets.get(ss[key_prefix + "preset"])
    if selected is not None and selected["description"]:
        st.caption(selected["description"])

    detections = ss.get(key_prefix + "preset_detections")
    if detections is not None:
        if "error" in detections:
            st.warning(f"Couldn't test the presets: {detections['error']}")
        else:
            if ss.pop(key_prefix + "preset_auto_selected", False) and selected is not None:
                st.success(f'Preset "{selected["name"]}" selected for this site. Check the selectors in Last Released Chapter.')
            if detections["working"]:
                lines = []
                for detection in detections["working"]:
                    chapter = detection["chapter"]
                    found = f"chapter {chapter['Chapter']}"
                    if chapter["URL"]:
                        found += f" ({chapter['URL']})"
                    lines.append(f"- **{detection['preset']['name']}**: {found}")
                st.markdown("Presets that work for this page:\n" + "\n".join(lines))
            elif not detections["domain_presets"]:
                st.info("No preset works for this page. Fill the selectors in Last Released Chapter.")

    with st.popover("Save selectors as preset", use_container_width=True):
        st.text_input("Preset name", key=key_prefix + "new_preset_name")
        st.text_input(
            "Domains",
            help="Comma-separated sites where this preset is selected automatically, like 'example.com'. Leave empty to only select it by hand.",
            key=key_prefix + "new_preset_domains",
        )
        st.button(
            "Save preset",
            key=key_prefix + "save_preset",
            type="primary",
            use_container_width=True,
            on_click=save_custom_manga_preset,
            args=(key_prefix,),
        )
        if selected is not None and not selected["builtin"]:
            st.button(
                f'Delete preset "{selected["name"]}"',
                key=key_prefix + "delete_preset",
                use_container_width=True,
                on_click=delete_custom_manga_preset,
                args=(key_prefix,),
            )

    if (message := ss.pop(key_prefix + "preset_message", None)) is not None:
        getattr(st, message[0])(message[1])


def show_add_custom_manga_form(multimanga_id: int = 0):
    key_prefix = "add_custom_manga_form_" if multimanga_id == 0 else "add_custom_manga_to_multimanga_form_"

    with st.container(border=False):
        st.text_input(
            "Manga Name (not optional)",
            placeholder="One Piece",
            key=key_prefix + "name",
        )

        st.text_input(
            "Manga URL",
            placeholder="https://randomsite.com/title/one-piece",
            key=key_prefix + "url",
            on_change=on_custom_manga_url_change,
            args=(key_prefix,),
        )

        show_custom_manga_presets(key_prefix)

        if multimanga_id == 0:
            st.selectbox(
                "Status",
                index=0,
                options=list(defaults.manga_status_options.keys())[
                    1:
                ],  # Exclude the "All" option
                format_func=lambda index: defaults.manga_status_options[index],
                key=key_prefix + "status",
            )

            with st.expander(
                "Last Read Chapter",
            ):
                st.text_input(
                    "Last Read Chapter",
                    placeholder="1000",
                    help="Can be a number or text.",
                    key=key_prefix + "last_read_chapter",
                )

                st.text_input(
                    "Chapter URL",
                    placeholder="https://randomsite.com/title/one-piece/chapter/1000",
                    help="Optional. Can provide only the chapter above.",
                    key=key_prefix + "last_read_chapter_url",
                )

        with st.expander(
            "Last Released Chapter",
        ):
            st.info(
                "Optional. Only fill if you want Mantium to automatically track new chapters."
            )
            with st.expander(
                "Chapter Name"
            ):
                st.text_input(
                    "Selector",
                    placeholder="css:div.chapter-list > a:nth-child(1) > div.chapter-title",
                    help="CSS or XPath selector to get the chapter name. E.g. 'css:div.chapter-list > a:nth-child(1) > div.chapter-title' or 'xpath://div[@class=\"chapter-list\"]//a[1]//div[@class=\"chapter-title\"]'. Leave empty to skip.",
                    key=key_prefix + "last_released_chapter_name_selector",
                )
                st.text_input(
                    "Attribute",
                    placeholder="href",
                    help="Element attribute to get the chapter name. E.g. 'href' for link. Leave empty to get the inner text.",
                    key=key_prefix + "last_released_chapter_name_attribute",
                )
                st.text_input(
                    "Regex",
                    placeholder="Chapter (\\d+)",
                    help="Regex to extract the chapter name. E.g. 'Chapter (\\d+)' to extract '100' from 'Chapter 100'. Leave empty to skip.",
                    key=key_prefix + "last_released_chapter_name_regex",
                )
                st.checkbox(
                    "Get First",
                    value=False,
                    help="Get the first match instead of the last match. Use it when the site lists the newest chapter first.",
                    key=key_prefix + "last_released_chapter_name_get_first",
                )
            with st.expander(
                "Chapter URL"
            ):
                st.text_input(
                    "Selector",
                    placeholder="css:div.chapter-list > a:nth-child(1)",
                    help="CSS or XPath selector to get the chapter URL. E.g. 'css:div.chapter-list > a:nth-child(1)' or 'xpath://div[@class=\"chapter-list\"]//a[1]'. Leave empty to skip.",
                    key=key_prefix + "last_released_chapter_url_selector",
                )
                st.text_input(
                    "Attribute",
                    placeholder="href",
                    help="Element attribute to get the chapter URL. E.g. 'href' for link. Leave empty to get the inner text.",
                    key=key_prefix + "last_released_chapter_url_attribute",
                )
                st.checkbox(
                    "Get First",
                    value=False,
                    help="Get the first match instead of the last match. Use it when the site lists the newest chapter first.",
                    key=key_prefix + "last_released_chapter_url_get_first",
                )

            st.checkbox(
                "Use Browser",
                value=False,
                help="Use a browser to fetch the manga page. Use this if the chapter info is loaded dynamically with JavaScript. Requires more resources and can take longer.",
                key=key_prefix + "last_released_chapter_use_browser",
            )

            if st.button(
                "Test selectors",
                key=key_prefix + "test_selectors",
                use_container_width=True,
                help="Get the last released chapter from the manga URL using the selectors above, without adding the manga.",
            ):
                test_custom_manga_selectors(
                    ss[key_prefix + "url"],
                    ss[key_prefix + "last_released_chapter_name_selector"],
                    ss[key_prefix + "last_released_chapter_name_attribute"],
                    ss[key_prefix + "last_released_chapter_name_regex"],
                    ss[key_prefix + "last_released_chapter_name_get_first"],
                    ss[key_prefix + "last_released_chapter_url_selector"],
                    ss[key_prefix + "last_released_chapter_url_attribute"],
                    ss[key_prefix + "last_released_chapter_url_get_first"],
                    ss[key_prefix + "last_released_chapter_use_browser"],
                )

        with st.expander(
            "Cover Image",
        ):
            st.info(
                "Provide only a cover image URL or a file. If neither are provided, Mantium will use a default cover image."
            )
            st.text_input(
                "Cover Image URL",
                placeholder="https://example.com/image.jpg",
                key=key_prefix + "cover_img_url",
            )
            st.file_uploader(
                "Upload Cover Image",
                type=["png", "jpg", "jpeg"],
                key=key_prefix + "cover_img_file",
            )

        if st.button(
            "Add",
            key=key_prefix + "add",
            use_container_width=True,
            type="primary",
        ):
            if ss[key_prefix + "name"] == "":
                st.warning("Provide a manga name")
            elif (
                ss.get(key_prefix + "last_read_chapter", "") == ""
                and ss.get(key_prefix + "last_read_chapter_url", "") != ""
            ):
                st.warning("If providing a last read chapter URL, also provide the chapter number")
            elif invalid_url_message := (
                get_invalid_url_message("manga URL", ss[key_prefix + "url"])
                or get_invalid_url_message("last read chapter URL", ss.get(key_prefix + "last_read_chapter_url", ""))
                or get_invalid_url_message("cover image URL", ss[key_prefix + "cover_img_url"])
            ):
                st.warning(invalid_url_message)
            else:
                if ss[key_prefix + "cover_img_file"] is not None:
                    cover_img = ss[key_prefix + "cover_img_file"].getvalue()
                else:
                    cover_img = None

                ss["add_manga_manga_to_add" if multimanga_id == 0 else "update_multimanga_mangas_add_manga_selected_manga"] = {
                    "name": ss[key_prefix + "name"],
                    "url": ss[key_prefix + "url"],
                    "status": ss.get(key_prefix + "status", ""),
                    "manga_internal_id": "",
                    "cover_img_url": ss[key_prefix + "cover_img_url"],
                    "cover_img": cover_img,
                    "last_released_chapter_name_selector": {
                        "selector": ss[key_prefix + "last_released_chapter_name_selector"],
                        "attribute": ss[key_prefix + "last_released_chapter_name_attribute"],
                        "regex": ss[key_prefix + "last_released_chapter_name_regex"],
                        "get_first": ss[key_prefix + "last_released_chapter_name_get_first"],
                    },
                    "last_released_chapter_url_selector": {
                        "selector": ss[key_prefix + "last_released_chapter_url_selector"],
                        "attribute": ss[key_prefix + "last_released_chapter_url_attribute"],
                        "get_first": ss[key_prefix + "last_released_chapter_url_get_first"],
                    },
                    "last_released_chapter_use_browser": ss[key_prefix + "last_released_chapter_use_browser"],
                }

                if multimanga_id == 0:
                    ss["add_manga_manga_to_add"]["last_read_chapter"] = {
                        "chapter": ss[key_prefix + "last_read_chapter"],
                        "url": ss[key_prefix + "last_read_chapter_url"],
                        "internal_id": "",
                    }

                    add_manga()
                else:
                    try:
                        with st.spinner("Adding manga to multimanga..."):
                            api_client = get_api_client()
                            api_client.add_manga_to_multimanga(
                                multimanga_id,
                                ss["update_multimanga_mangas_add_manga_selected_manga"]["name"],
                                ss["update_multimanga_mangas_add_manga_selected_manga"]["url"],
                                "",
                                ss["update_multimanga_mangas_add_manga_selected_manga"]["cover_img_url"],
                                ss["update_multimanga_mangas_add_manga_selected_manga"]["cover_img"],
                                ss["update_multimanga_mangas_add_manga_selected_manga"]["last_released_chapter_name_selector"]["selector"],
                                ss["update_multimanga_mangas_add_manga_selected_manga"]["last_released_chapter_name_selector"]["attribute"],
                                ss["update_multimanga_mangas_add_manga_selected_manga"]["last_released_chapter_name_selector"]["regex"],
                                ss["update_multimanga_mangas_add_manga_selected_manga"]["last_released_chapter_name_selector"]["get_first"],
                                ss["update_multimanga_mangas_add_manga_selected_manga"]["last_released_chapter_url_selector"]["selector"],
                                ss["update_multimanga_mangas_add_manga_selected_manga"]["last_released_chapter_url_selector"]["attribute"],
                                ss["update_multimanga_mangas_add_manga_selected_manga"]["last_released_chapter_url_selector"]["get_first"],
                                ss["update_multimanga_mangas_add_manga_selected_manga"]["last_released_chapter_use_browser"],
                            )
                    except APIException as e:
                        resp_text = str(e.response_text).lower()
                        if user_message := get_api_user_message(e):
                            st.warning(user_message)
                        elif "manga already exists in db" in resp_text:
                            st.warning("Manga already exists")
                        elif "source not found" in resp_text:
                            st.warning("No source site for this manga")
                        elif "invalid manga url" in resp_text:
                            st.warning("Invalid URL")
                        elif "manga not found in source" in resp_text:
                            st.warning("Manga not found")
                        elif "one of the manga attributes were not found in source" in resp_text:
                            st.warning("Manga not found")
                        elif "source" in str(e).lower() and "is not allowed" in str(e).lower():
                            st.warning("Not allowed to add mangas from this source")
                        else:
                            logger.exception(e)
                            st.error(f"Error while adding manga to multimanga: {get_api_error_detail(e)}")
                    else:
                        ss["update_manga_success_message"] = "Manga added successfully"
                        st.rerun()

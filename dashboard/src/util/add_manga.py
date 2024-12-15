from typing import Any

import src.util.defaults as defaults
import streamlit as st
from src.api.api_client import get_api_client
from src.exceptions import APIException
from src.util.util import (
    centered_container,
    get_logger,
    get_relative_time,
    get_updated_at_datetime,
    tagger,
)
from streamlit import session_state as ss
from streamlit_javascript import st_javascript

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
    ss[base_key + "_comick"] = {}
    ss[base_key + "_mangaplus"] = {}
    ss[base_key + "_mangahub"] = {}
    ss[base_key + "_mangaupdates"] = {}
    ss[base_key + "_rawkuma"] = {}
    ss[base_key + "_klmanga"] = {}
    ss[base_key + "_jmanga"] = {}
    ss["add_manga_search_go_back_to_tab"] = 0

    if form_type == "url":

        @st.dialog("Add Manga")
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

        @st.dialog("Add Manga", width="large")
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

        @st.dialog("Add Custom Manga")
        def show():
            ss["is_dialog_open"] = True
            e = st.empty()
            if ss.get("add_custom_manga_manga_to_add", None) is not None:
                with e:
                    add_custom_manga()
                    del ss["add_custom_manga_manga_to_add"]
            else:
                with e.container():
                    show_add_custom_manga_form()

    else:
        st.stop()

    show()


def add_manga():
    api_client = get_api_client()
    ex = None
    with st.spinner("Adding manga..."):
        try:
            api_client.add_multimanga(
                ss["add_manga_manga_to_add"]["manga_url"],
                ss["add_manga_manga_to_add"]["status"],
                ss["add_manga_manga_to_add"]["manga_internal_id"],
                ss["add_manga_manga_to_add"]["chapter"],
                ss["add_manga_manga_to_add"]["chapter_url"],
                ss["add_manga_manga_to_add"]["chapter_internal_id"],
            )
        except Exception as e:
            ex = e

    ss["add_manga_search_selected_manga"] = None
    ss["add_manga_form_url"] = ""
    if ex is not None:
        if (
            "multimang added to DB, but error executing integrations:".lower()
            in str(ex).lower()
        ):
            logger.warning(ex)
            ss["add_manga_warning_message"] = (
                "Manga added to DB, but couldn't add it to at least one integration"
            )
            st.rerun()
        elif "manga already exists in DB".lower() in str(ex).lower():
            st.warning("Manga already in Mantium")
        else:
            logger.exception(ex)
            st.error("Error while adding manga")
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
                "manga_url": manga_url,
                "status": ss.add_manga_form_status,
                "manga_internal_id": manga_internal_id,
                "chapter": chapter.get("Chapter", ""),
                "chapter_url": chapter.get("URL", ""),
                "chapter_internal_id": chapter.get("InternalID", ""),
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
            if (
                "error while getting source: source '" in resp_text
                and "not found" in resp_text
            ):
                st.warning("No source site for this manga")
            elif (
                "manga doesn't have and id or url" in resp_text
                or "invalid uri for request" in resp_text
            ):
                st.warning("Invalid URL")
            elif "manga not found in source" in resp_text:
                st.warning("Manga not found")
            else:
                logger.exception(e)
                st.error("Error while getting manga chapters.")
                st.stop()
        else:
            show_bottom_add_manga_form(ss.add_manga_form_url, "")


def show_add_manga_form_search():
    api_client = get_api_client()
    container = st.empty()
    if ss.get("add_manga_search_selected_manga", None) is not None:
        with container:
            try:
                with st.spinner("Getting manga chapters..."):
                    ss["add_manga_chapter_options"] = (
                        api_client.get_cached_manga_chapters(
                            -1,
                            ss["add_manga_search_selected_manga"]["URL"],
                            ss["add_manga_search_selected_manga"]["InternalID"],
                        )
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
            match ss["add_manga_search_selected_manga"]["Source"]:
                case "mangadex":
                    ss["add_manga_search_go_back_to_tab"] = 0
                case "comick":
                    ss["add_manga_search_go_back_to_tab"] = 1
                case "mangaplus":
                    ss["add_manga_search_go_back_to_tab"] = 2
                case "mangahub":
                    ss["add_manga_search_go_back_to_tab"] = 3
                case "mangaupdates":
                    ss["add_manga_search_go_back_to_tab"] = 4
                case "rawkuma":
                    ss["add_manga_search_go_back_to_tab"] = 5
                case "klmanga":
                    ss["add_manga_search_go_back_to_tab"] = 6
                case "jmanga":
                    ss["add_manga_search_go_back_to_tab"] = 7
            ss["add_manga_search_selected_manga"] = None

        st.button("Back", use_container_width=True, on_click=on_click)
    else:
        with container:
            (
                mangadex_tab,
                comick_tab,
                mangaplus_tab,
                mangahub_tab,
                mangaupdates_tab,
                rawkuma_tab,
                klmanga_tab,
                jmanga_tab,
            ) = st.tabs(
                [
                    "Mangadex",
                    "Comick",
                    "Mangaplus",
                    "Mangahub",
                    "MangaUpdates",
                    "RawKuma",
                    "KLManga",
                    "JManga",
                ]
            )

            # if change key_to_save_manga, also change it in func show_dialogs in the 01_?.py main file
            button_name, key_to_save_manga = "Select", "add_manga_search_selected_manga"

            with mangadex_tab:
                show_search_manga_term_form(
                    "https://mangadex.org", button_name, key_to_save_manga
                )
            with comick_tab:
                show_search_manga_term_form(
                    "https://comick.io", button_name, key_to_save_manga
                )
            with mangaplus_tab:
                show_search_manga_term_form(
                    "https://mangaplus.shueisha.co.jp", button_name, key_to_save_manga
                )
            with mangahub_tab:
                show_search_manga_term_form(
                    "https://mangahub.io", button_name, key_to_save_manga
                )
            with mangaupdates_tab:
                show_search_manga_term_form(
                    "https://mangaupdates.com", button_name, key_to_save_manga
                )
            with rawkuma_tab:
                show_search_manga_term_form(
                    "https://rawkuma.com", button_name, key_to_save_manga
                )
            with klmanga_tab:
                show_search_manga_term_form(
                    "https://klmanga.rs", button_name, key_to_save_manga
                )
            with jmanga_tab:
                show_search_manga_term_form(
                    "https://jmanga.is", button_name, key_to_save_manga
                )

        tab_index = ss["add_manga_search_go_back_to_tab"]
        js = f"""window.parent.document.querySelectorAll('button[data-baseweb="tab"]')[{tab_index}].click();"""
        st_javascript(js)
        js = """window.parent.document.querySelectorAll('div:has(> iframe[title="streamlit_javascript.streamlit_javascript"])').forEach(div => div.style.display = 'none');"""
        st_javascript(js)


def show_search_manga_term_form(
    source_site_url: str, button_name: str, key_to_save_manga: str
):
    """Show search manga term form.

    Args:
        source_site_url (str): The source site URL to search for manga.
        button_name (str): The name of the button to select a manga.
        key_to_save_manga (str): The key to save the selected manga in streamlit.session_state.
    """
    api_client = get_api_client()
    search_results_key = f"{key_to_save_manga}_search_results_{source_site_url.split('//')[1].split('.')[0]}"
    search_term_key = f"{key_to_save_manga}_search_term_{source_site_url.split('//')[1].split('.')[0]}"

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
                    ss["settings_search_results_limit"],
                    source_site_url,
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
        show_search_result_mangas(
            st.columns(2), results, button_name, key_to_save_manga
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
    for manga in mangas:
        if col_index == len(cols_list):
            col_index = 0
        with cols_list[col_index]:
            with st.container(border=True, height=manga_container_height):
                with centered_container("center_container"):
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
                <a class="manga_header" href="{manga["URL"]}" target="_blank">{manga["Name"]}</a>
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
            manga["LastChapter"] if manga["LastChapter"] not in ("", "0") else "N/A",
        )
    else:
        last_chapter = f"""
            <a href="{manga["LastChapterURL"]}" target="_blank" style="text-decoration: none; color: {defaults.chapter_link_tag_text_color}">
                <span>{manga["LastChapter"] if manga["LastChapter"] not in ("", "0") else "N/A"}</span>
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


def show_add_custom_manga_form():
    api_client = get_api_client()

    with st.form(key="add_custom_manga_form", border=False):
        st.text_input(
            "Manga Name (not optional)",
            placeholder="One Piece",
            key="add_custom_manga_form_name",
        )

        st.text_input(
            "Manga URL",
            placeholder="https://randomsite.com/title/one-piece",
            key="add_custom_manga_form_url",
        )

        st.selectbox(
            "Status",
            index=0,
            options=list(defaults.manga_status_options.keys())[
                1:
            ],  # Exclude the "All" option
            format_func=lambda index: defaults.manga_status_options[index],
            key="add_custom_manga_form_status",
        )

        with st.expander(
            "Next Chapter",
        ):
            st.text_input(
                "New Chapter to Read",
                placeholder="1000",
                help="Can be a number or text",
                key="add_custom_manga_form_chapter",
            )

            st.text_input(
                "Chapter URL",
                placeholder="https://randomsite.com/title/one-piece/chapter/1000",
                key="add_custom_manga_form_chapter_url",
            )

            st.checkbox(
                "No more chapters available",
                help="Check this if there are no more chapters available. You can change it later.",
                key="add_custom_manga_form_no_more_chapters",
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
                key="add_custom_manga_form_cover_img_url",
            )
            st.file_uploader(
                "Upload Cover Image",
                type=["png", "jpg", "jpeg"],
                key="add_custom_manga_form_cover_img_file",
            )

        if st.form_submit_button(
            "Add",
            use_container_width=True,
            type="primary",
        ):
            if ss.add_custom_manga_form_name == "":
                st.warning("Provide a manga name")
            elif (
                ss.add_custom_manga_form_chapter == ""
                and ss.add_custom_manga_form_chapter_url != ""
            ):
                st.warning("If providing a chapter URL, also provide a chapter number")
            else:
                if ss.add_custom_manga_form_cover_img_file is not None:
                    cover_img = ss.add_custom_manga_form_cover_img_file.getvalue()
                else:
                    cover_img = None

                ss["add_custom_manga_manga_to_add"] = {
                    "name": ss.add_custom_manga_form_name,
                    "url": ss.add_custom_manga_form_url,
                    "status": ss.add_custom_manga_form_status,
                    "manga_has_more_chapters": not ss.add_custom_manga_form_no_more_chapters,
                    "cover_img_url": ss.add_custom_manga_form_cover_img_url,
                    "cover_img": cover_img,
                    "next_chapter": {
                        "chapter": ss.add_custom_manga_form_chapter,
                        "url": ss.add_custom_manga_form_chapter_url,
                    },
                }
                st.rerun(scope="fragment")


def add_custom_manga():
    api_client = get_api_client()
    ex = None
    with st.spinner("Adding manga..."):
        try:
            api_client.add_custom_manga(
                ss["add_custom_manga_manga_to_add"]["name"],
                ss["add_custom_manga_manga_to_add"]["url"],
                ss["add_custom_manga_manga_to_add"]["status"],
                ss["add_custom_manga_manga_to_add"]["manga_has_more_chapters"],
                ss["add_custom_manga_manga_to_add"]["cover_img_url"],
                ss["add_custom_manga_manga_to_add"]["cover_img"],
                ss["add_custom_manga_manga_to_add"]["next_chapter"]["chapter"],
                ss["add_custom_manga_manga_to_add"]["next_chapter"]["url"],
            )
        except Exception as e:
            ex = e

    if ex is not None:
        resp_text = str(ex).lower()
        if "manga already exists in DB".lower() in resp_text:
            st.warning("Manga already in Mantium")
        if (
            "duplicate key value violates unique constraint" in resp_text
            and "chapters_pkey" in resp_text
        ):
            st.warning("Next chapter URL already in Mantium")
        else:
            logger.exception(ex)
            st.error("Error while adding manga")
    else:
        ss["add_manga_success_message"] = "Manga added successfully"
        ss["add_manga_search_selected_manga"] = None
        st.rerun()

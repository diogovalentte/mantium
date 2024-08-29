import base64
import logging
from datetime import datetime
from io import BytesIO
from typing import Any

import streamlit as st
from PIL import Image
from src.api.api_client import get_api_client
from src.exceptions import APIException
from src.util import (centered_container, fix_streamlit_index_html,
                      get_relative_time, tagger)
from streamlit import session_state as ss
from streamlit_extras.stylable_container import stylable_container
from streamlit_javascript import st_javascript

st.set_page_config(
    page_title="Mantium",
    page_icon="📖",
    layout="wide",
)

logging.basicConfig(
    encoding="utf-8",
    level=logging.INFO,
    format="%(asctime)s :: %(levelname)-8s :: %(name)s :: %(message)s",
)
logger = logging.getLogger()


class MainDashboard:
    def __init__(self, api_client):
        self.api_client = api_client
        self.manga_status_options = {
            0: "📚 All",
            1: "📖 Reading",
            2: "✅ Completed",
            3: "🚧 On Hold",
            4: "❌ Dropped",
            5: "📅 Plan to Read",
        }
        self.status_filter_key = 1
        self.sort_options = [
            "Name",
            "Unread",
            "Last Read",
            "Chapters Released",
            "Released Chapter Date",
        ]
        self.sort_option_index = 1
        self.chapter_link_tag_background_color = "rgb(219 233 254)"
        self.chapter_link_tag_text_color = "rgb(59 130 246)"

    def show(self):
        self.check_dashboard_error()

        self.sidebar()

        mangas = self.api_client.get_mangas()
        mangas = [
            manga
            for manga in mangas
            if ss.get(
                "status_filter",
                self.status_filter_key,
            )
            == 0
            or manga["Status"]
            == ss.get(
                "status_filter",
                self.status_filter_key,
            )
        ]
        if ss.get("search_manga", "") != "":
            mangas = [
                manga
                for manga in mangas
                if ss.get("search_manga", "").upper() in manga["Name"].upper()
            ]

        mangas = self.api_client.sort_mangas(
            mangas,
            ss.get("mangas_sort", self.sort_options[self.sort_option_index]),
            ss.get("mangas_sort_reverse", False),
        )
        self.show_mangas(st.columns(ss["configs_columns_number"]), mangas)

        if "system_last_update_time" not in ss:
            ss["system_last_update_time"] = self.api_client.check_for_updates()

        @st.experimental_fragment(run_every=5)
        def check_for_updates():
            last_update = self.api_client.check_for_updates()
            if last_update != ss["system_last_update_time"]:
                ss["system_last_update_time"] = last_update
                st.rerun()

        check_for_updates()

        if (
            ss.get("manga_updated_success_message", "") != ""
            or ss.get("manga_updated_error_message", "") != ""
            or ss.get("manga_update_warning_message", "") != ""
        ):

            @st.experimental_dialog("Update Manga")
            def show_update_manga_dialog():
                if ss.get("manga_updated_success_message", "") != "":
                    st.success(ss["manga_updated_success_message"])
                if ss.get("manga_updated_error_message", "") != "":
                    st.error(ss["manga_updated_error_message"])
                if ss.get("manga_update_warning_message", "") != "":
                    st.warning(ss["manga_update_warning_message"])

            show_update_manga_dialog()

            ss["manga_updated_success_message"] = ""
            ss["manga_updated_error"] = ""
            ss["manga_update_warning_message"] = ""

    def sidebar(self) -> None:
        with st.sidebar:
            self.show_background_error()

            st.text_input("Search", key="search_manga")

            def status_filter_callback():
                self.status_filter_key = ss.status_filter

            st.selectbox(
                "Filter Status",
                self.manga_status_options,
                index=self.sort_option_index,
                on_change=status_filter_callback,
                format_func=lambda index: self.manga_status_options[index],
                key="status_filter",
            )

            def sort_callback():
                self.sort_option_index = ss.mangas_sort

            st.selectbox(
                "Sort By",
                self.sort_options,
                index=self.status_filter_key,
                on_change=sort_callback,
                key="mangas_sort",
            )

            st.toggle("Reverse Sort", key="mangas_sort_reverse")
            st.divider()

            with st.expander("Add Manga"):
                if st.button(
                    "Search by Name",
                    type="primary",
                    use_container_width=True,
                ):
                    ss["manga_add_success_message"] = False
                    ss["manga_add_warning_message"] = ""
                    ss["manga_add_error_message"] = ""
                    if ss.get("add_manga_chapter_options", None) is not None:
                        del ss["add_manga_chapter_options"]
                    if ss.get("add_manga_search_selected_manga", None) is not None:
                        del ss["add_manga_search_selected_manga"]
                    ss["add_manga_search_results_mangadex"] = {}
                    ss["add_manga_search_results_comick"] = {}
                    ss["add_manga_search_results_mangaplus"] = {}
                    ss["add_manga_search_results_mangahub"] = {}
                    ss["add_manga_search_go_back_to_tab"] = 0

                    @st.experimental_dialog("Search Manga", width="large")
                    def show_add_manga_form_dialog():
                        self.show_add_manga_form_search()

                    show_add_manga_form_dialog()

                if st.button("Add using URL", type="primary", use_container_width=True):
                    ss["manga_add_success_message"] = False
                    ss["manga_add_warning_message"] = ""
                    ss["manga_add_error_message"] = ""
                    if ss.get("add_manga_chapter_options", None) is not None:
                        del ss["add_manga_chapter_options"]

                    @st.experimental_dialog("Add Manga Using URL")
                    def show_add_manga_form_dialog():
                        self.show_add_manga_form_url()

                    show_add_manga_form_dialog()
                if ss.get("manga_add_success_message", False):
                    st.success("Manga added successfully")
                elif ss.get("manga_add_warning_message", "") != "":
                    st.warning(ss["manga_add_warning_message"])
                elif ss.get("manga_add_error_message", "") != "":
                    st.warning(ss["manga_add_error_message"])

            st.divider()
            self.show_configs()

    def show_background_error(self):
        if ss["configs_show_background_error_warning"]:
            last_background_error = self.api_client.get_last_background_error()
            if last_background_error["message"] != "":
                with st.expander("An error occurred in the background!", expanded=True):
                    logger.error(
                        f"Background error: {last_background_error['message']}"
                    )
                    st.info(f"Time: {last_background_error['time']}")

                    @st.experimental_dialog(
                        "Last Background Error Message", width="large"
                    )
                    def show_error_message_dialog():
                        st.write(last_background_error["message"])

                    if st.button(
                        "See error",
                        type="primary",
                        help="See error message",
                        use_container_width=True,
                    ):
                        show_error_message_dialog()
                    with stylable_container(
                        key="highlight_manga_delete_button",
                        css_styles="""
                            button {
                                background-color: red;
                                color: white;
                            }
                        """,
                    ):
                        st.button(
                            "Delete Error",
                            use_container_width=True,
                            help="Delete the last background error",
                            on_click=self.api_client.delete_last_background_error,
                        )
                st.divider()

    def show_mangas(self, cols_list: list, mangas: list[dict[str, Any]]):
        """Show mangas in the cols_list columns.

        Args:
            cols_list (list): A list of streamlit.columns.
            mangas (dict): A list of mangas.
        """
        col_index = 0
        for manga in mangas:
            if col_index == len(cols_list):
                col_index = 0
            with cols_list[col_index]:
                with st.container(border=True):
                    with centered_container("center_container"):
                        self.show_manga_dashboard(manga)
            col_index += 1

    def show_manga_dashboard(self, manga: dict[str, Any]):
        unread = (
            manga["LastReadChapter"]["Chapter"]
            != manga["LastReleasedChapter"]["Chapter"]
        )

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
                h1.manga_header > div > span {
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

                @keyframes pulse {
                    0% {
                        color: white; /* Start color */
                    }
                    100% {
                        color: #04c9b7; /* End color */
                    }
                }
            </style>
        """
        st.markdown(improve_headers, unsafe_allow_html=True)
        st.markdown(
            f"""<h1
                class="manga_header" style='text-align: center; {"animation: pulse 2s infinite alternate;" if unread else ""} margin-top: {margin}px; margin-bottom: {margin}px; font-size: {font_size}px;'>
                    <a class="manga_header" href="{manga["URL"]}" target="_blank">{manga["Name"]}</a>
                </h1>
            """,
            unsafe_allow_html=True,
        )

        if manga["CoverImg"] is not None:
            img_bytes = base64.b64decode(manga["CoverImg"])
            img = BytesIO(img_bytes)
            if not manga["CoverImgResized"]:
                img = Image.open(img)
                img = img.resize((250, 355))
            st.image(img)
        elif manga["CoverImgURL"] != "":
            st.markdown(
                f"""<img src="{manga["CoverURL"]}" width="250" height="355"/>""",
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

        if ss.get("status_filter", 1) == 0:
            st.write(
                f'**Status**: <span style="float: right;">{self.manga_status_options[manga["Status"]]}</span>',
                unsafe_allow_html=True,
            )

        chapter_tag_content = f"""
            <a href="{{}}" target="_blank" style="text-decoration: none; color: {self.chapter_link_tag_text_color}">
                <span>{{}}</span>
            </a>
        """

        chapter = chapter_tag_content.format(
            manga["LastReleasedChapter"]["URL"],
            f'Ch. {manga["LastReleasedChapter"]["Chapter"]}'
            if manga["LastReleasedChapter"]["Chapter"] != ""
            else "N/A",
        )
        release_date = (
            manga["LastReleasedChapter"]["UpdatedAt"]
            if manga["LastReleasedChapter"]["UpdatedAt"] != datetime(1970, 1, 1)
            else "N/A"
        )
        if release_date != "N/A":
            relative_release_date = get_relative_time(release_date)
        else:
            relative_release_date = release_date

        tagger(
            "<strong>Last Released Chapter:</strong>",
            chapter,
            self.chapter_link_tag_background_color,
            "float: right;",
        )
        st.caption(
            f'**Release Date**: <span style="float: right;" title="{release_date}">{relative_release_date}</span>',
            unsafe_allow_html=True,
        )

        chapter = chapter_tag_content.format(
            manga["LastReadChapter"]["URL"],
            f'Ch. {manga["LastReadChapter"]["Chapter"]}'
            if manga["LastReadChapter"]["Chapter"] != ""
            else "N/A",
        )
        read_date = (
            manga["LastReadChapter"]["UpdatedAt"]
            if manga["LastReadChapter"]["UpdatedAt"] != datetime(1970, 1, 1)
            else "N/A"
        )
        if read_date != "N/A":
            relative_read_date = get_relative_time(read_date)
        else:
            relative_read_date = read_date

        tagger(
            "<strong>Last Read Chapter:</strong>",
            chapter,
            self.chapter_link_tag_background_color,
            "float: right;",
        )
        st.caption(
            f'**Read Date**: <span style="float: right;" title="{read_date}">{relative_read_date}</span>',
            unsafe_allow_html=True,
        )

        c1, c2 = st.columns(2)
        with c1:

            def set_last_read():
                if (
                    manga.get("LastReadChapter", {}).get("Chapter")
                    != manga["LastReleasedChapter"]["Chapter"]
                ):
                    self.api_client.update_manga_last_read_chapter(
                        manga["ID"],
                        manga["URL"],
                    )

            st.button(
                "Set last read",
                use_container_width=True,
                on_click=set_last_read,
                key=f"set_last_read_{manga['ID']}",
                disabled=not unread,
            )
        with c2:
            if st.button(
                "Highlight",
                use_container_width=True,
                type="primary",
                key=f"highlight_{manga['ID']}",
            ):

                @st.experimental_dialog(manga["Name"])
                def show_highlighted_manga_dialog():
                    self.show_highlighted_manga(manga)

                show_highlighted_manga_dialog()

    def show_highlighted_manga(self, manga: dict[str, Any]):
        with st.spinner("Getting manga chapters..."):
            try:
                ss["update_manga_chapter_options"] = get_manga_chapters(
                    manga["ID"], manga["URL"]
                )
            except APIException as e:
                logger.exception(e)
                st.error("Error while getting manga chapters")
                ss["update_manga_chapter_options"] = []

        with st.form(key="update_manga_form", border=False):
            st.selectbox(
                "Status",
                index=manga["Status"] - 1,
                options=list(self.manga_status_options.keys())[
                    1:
                ],  # Exclude the "All" option
                format_func=lambda index: self.manga_status_options[index],
                key="update_manga_form_status",
            )

            if manga["LastReadChapter"]["Chapter"] != "":
                try:
                    last_read_chapter_idx = list(
                        map(
                            lambda chapter: chapter["Chapter"],
                            ss["update_manga_chapter_options"],
                        )
                    ).index(manga["LastReadChapter"]["Chapter"])
                except ValueError as e:
                    st.warning(
                        "Last read chapter not found in the manga chapters. Select it again."
                    )
                    logger.warning(e)
                    last_read_chapter_idx = None
            else:
                last_read_chapter_idx = 0
            st.selectbox(
                "Last Read Chapter",
                index=last_read_chapter_idx,
                options=ss["update_manga_chapter_options"],
                format_func=lambda chapter: f"Ch. {chapter['Chapter']} --- {get_relative_time(self.api_client.get_updated_at_datetime(chapter['UpdatedAt']))}",
                key="update_manga_form_chapter",
            )

            if (
                ss.get("update_manga_chapter_options") is not None
                and len(ss.get("update_manga_chapter_options", [])) < 1
            ):
                st.warning(
                    "Manga has no released chapters. You still can update the other fields."
                )

            with st.expander(
                "Update Cover Image",
            ):
                st.info(
                    "By default, the cover image is fetched from the source site, but you can manually provide an image URL or upload a file."
                )
                st.text_input(
                    "Cover Image URL",
                    placeholder="https://example.com/image.jpg",
                    key="update_manga_form_cover_img_url",
                )
                st.file_uploader(
                    "Upload Cover Image",
                    type=["png", "jpg", "jpeg"],
                    key="update_manga_form_cover_img_upload",
                )
                st.divider()
                st.info(
                    "If you manually changed the cover image and want to go back and let the Mantium fetch the cover image from the source site, check the box below."
                )
                st.checkbox(
                    "Get cover image from source site.",
                    key="update_manga_form_get_cover_img_from_source",
                )

            if st.form_submit_button(
                "Update Manga",
                use_container_width=True,
                type="primary",
            ):
                try:
                    status = ss.update_manga_form_status
                    if status != manga["Status"]:
                        self.api_client.update_manga_status(status, manga["ID"])

                    chapter = ss.update_manga_form_chapter
                    if chapter is not None and (
                        manga["LastReadChapter"] is None
                        or chapter["URL"] != manga["LastReadChapter"]["URL"]
                        or chapter["Chapter"] != manga["LastReadChapter"]["Chapter"]
                    ):
                        self.api_client.update_manga_last_read_chapter(
                            manga["ID"],
                            manga["URL"],
                            chapter["Chapter"],
                            chapter["URL"],
                        )

                    cover_url = ss.update_manga_form_cover_img_url
                    cover_upload = (
                        ss.update_manga_form_cover_img_upload.getvalue()
                        if ss.update_manga_form_cover_img_upload
                        else None
                    )
                    get_cover_img_from_source = (
                        ss.update_manga_form_get_cover_img_from_source
                    )

                    values_count = sum(
                        [
                            bool(cover_url),
                            bool(cover_upload),
                            get_cover_img_from_source,
                        ]
                    )

                    match values_count:
                        case 0:
                            pass
                        case 1:
                            if cover_url != "" or cover_upload is not None:
                                self.api_client.update_manga_cover_img(
                                    manga["ID"],
                                    manga["URL"],
                                    cover_img_url=cover_url,
                                    cover_img=cover_upload if cover_upload else b"",
                                )
                            elif get_cover_img_from_source:
                                self.api_client.update_manga_cover_img(
                                    manga["ID"],
                                    manga["URL"],
                                    get_cover_img_from_source=get_cover_img_from_source,
                                )
                        case _:
                            ss["manga_update_warning_message"] = (
                                "To update the cover image, provide either an URL, upload a file, or check the box to get the image from the source site. The other fields were updated successfully."
                            )
                            st.rerun()
                    ss["manga_updated_success_message"] = "Manga updated successfully"
                    st.rerun()
                except APIException as e:
                    logger.exception(e)
                    ss["manga_updated_error"] = "Error while updating manga."
                    st.rerun()

        def delete_manga_btn_callback():
            self.api_client.delete_manga(manga["ID"])

        with stylable_container(
            key="highlight_manga_delete_button",
            css_styles="""
                button {
                    background-color: red;
                    color: white;
                }
            """,
        ):
            st.button(
                "Delete Manga",
                on_click=delete_manga_btn_callback,
                use_container_width=True,
                key="delete_manga_btn",
            )

    def show_add_manga_form_search(self):
        container = st.empty()
        if ss.get("add_manga_search_selected_manga", None) is not None:
            with container:
                try:
                    with st.spinner("Getting manga chapters..."):
                        ss["add_manga_chapter_options"] = get_manga_chapters(
                            -1, ss["add_manga_search_selected_manga"]["URL"]
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
                    else:
                        logger.exception(e)
                        st.error("Error while getting manga chapters.")

                self.show_add_manga_form(ss["add_manga_search_selected_manga"]["URL"])

            def on_click():
                match ss["add_manga_search_selected_manga"]["Source"]:
                    case "mangadex.org":
                        ss["add_manga_search_go_back_to_tab"] = 0
                    case "comick.xyz":
                        ss["add_manga_search_go_back_to_tab"] = 1
                    case "mangaplus.shueisha.co.jp":
                        ss["add_manga_search_go_back_to_tab"] = 2
                    case "mangahub.io":
                        ss["add_manga_search_go_back_to_tab"] = 3
                ss["add_manga_search_selected_manga"] = None

            st.button("Back", use_container_width=True, on_click=on_click)
        else:
            with container:
                mangadex_tab, comick_tab, mangaplus_tab, mangahub_tab = st.tabs(
                    ["Mangadex", "Comick", "Mangaplus", "Mangahub"]
                )

                with mangadex_tab:
                    self.show_search_manga_term_form("https://mangadex.org")
                with comick_tab:
                    self.show_search_manga_term_form("https://comick.io")
                with mangaplus_tab:
                    self.show_search_manga_term_form("https://mangaplus.shueisha.co.jp")
                with mangahub_tab:
                    self.show_search_manga_term_form("https://mangahub.io")

            tab_index = ss["add_manga_search_go_back_to_tab"]
            js = f"""window.parent.document.querySelectorAll('button[data-baseweb="tab"]')[{tab_index}].click();"""
            st_javascript(js)
            js = """window.parent.document.querySelectorAll('div:has(> iframe[title="streamlit_javascript.streamlit_javascript"])').forEach(div => div.style.display = 'none');"""
            st_javascript(js)

    def show_search_manga_term_form(self, source_site_url: str):
        search_results_key = (
            f"add_manga_search_results_{source_site_url.split('//')[1].split('.')[0]}"
        )
        search_term_key = f"search_manga_{source_site_url.split('//')[1].split('.')[0]}"

        term = st.text_input(
            "Term to Search",
            value=ss[search_term_key]
            if ss.get(search_term_key, "") != ""
            else ss[search_results_key].get("term", ""),
            key=search_term_key,
        )

        if term == "":
            ss[search_results_key]["term"] = term
            return
        elif ss[search_results_key].get("term", "") == term:
            results = ss[search_results_key].get("results", [])
        else:
            with st.spinner("Searching..."):
                results = self.api_client.search_manga(
                    term, ss["configs_search_results_limit"], source_site_url
                )
                ss[search_results_key]["results"] = results
            ss[search_results_key]["term"] = term

        if len(results) == 0:
            st.warning("No results found.")
        else:
            self.show_search_result_mangas(st.columns(2), results)
            st.info(
                "Did not find the manga you were looking for? Try another source site or using the URL directly."
            )

    def show_search_result_mangas(self, cols_list: list, mangas: list[dict[str, Any]]):
        """Show search result mangas in the cols_list columns.

        Args:
            cols_list (list): A list of streamlit.columns.
            mangas (dict): A list of search result mangas.
        """
        manga_container_height = 660
        col_index = 0
        for manga in mangas:
            if col_index == len(cols_list):
                col_index = 0
            with cols_list[col_index]:
                with st.container(border=True, height=manga_container_height):
                    with centered_container("center_container"):
                        self.show_search_result_manga(manga)
            col_index += 1

    def show_search_result_manga(self, manga: dict[str, Any]):
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
            <span style="color: {self.chapter_link_tag_text_color}">{{}}</span>
        """

        status = tag_content_format.format(
            manga["Status"].capitalize() if manga["Status"] != "" else "N/A",
        )
        tagger(
            "<strong>Status:</strong>",
            status,
            self.chapter_link_tag_background_color,
            "float: right;",
        )

        year = tag_content_format.format(
            manga["Year"] if manga["Year"] not in ("", "0", 0) else "N/A",
        )
        tagger(
            "<strong>Year:</strong>",
            year,
            self.chapter_link_tag_background_color,
            "float: right;",
        )

        last_chapter = tag_content_format.format(
            manga["LastChapter"] if manga["LastChapter"] not in ("", "0") else "N/A",
        )
        tagger(
            "<strong>Last Chapter:</strong>",
            last_chapter,
            self.chapter_link_tag_background_color,
            "float: right;",
        )

        st.caption(manga["Description"])

        def on_click():
            ss["add_manga_search_selected_manga"] = manga

        st.button(
            "Select",
            type="primary",
            use_container_width=True,
            on_click=on_click,
            key=f"add_manga_search_select_button_{manga['URL']}",
        )

    def show_add_manga_form_url(self):
        manga_url = st.text_input(
            "Manga URL",
            placeholder="https://mangahub.io/manga/one-piece",
            key="add_manga_form_url",
        )

        if manga_url:
            try:
                with st.spinner("Getting manga chapters..."):
                    ss["add_manga_chapter_options"] = get_manga_chapters(-1, manga_url)
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
                else:
                    logger.exception(e)
                    st.error("Error while getting manga chapters.")

        self.show_add_manga_form(ss.add_manga_form_url)

    def show_add_manga_form(self, manga_url: str):
        with st.form(key="add_manga_form", border=False, clear_on_submit=True):
            st.selectbox(
                "Status",
                index=0,
                options=list(self.manga_status_options.keys())[
                    1:
                ],  # Exclude the "All" option
                format_func=lambda index: self.manga_status_options[index],
                key="add_manga_form_status",
            )

            st.selectbox(
                "Last Read Chapter",
                options=ss.get("add_manga_chapter_options", []),
                key="add_manga_form_chapter",
                format_func=lambda chapter: f"Ch. {chapter['Chapter']} --- {get_relative_time(self.api_client.get_updated_at_datetime(chapter['UpdatedAt']))}"
                if chapter is not None
                else "N/A",
            )

            if (
                ss.get("add_manga_chapter_options") is not None
                and len(ss.get("add_manga_chapter_options", [])) < 1
            ):
                st.warning("Manga has no released chapters. You still can add it.")

            def add_manga_callback():
                ss["add_manga_manga_to_add"] = {
                    "manga_url": manga_url,
                    "status": ss.add_manga_form_status,
                    "chapter": ss.add_manga_form_chapter["Chapter"]
                    if ss.add_manga_form_chapter is not None
                    else "",
                    "chapter_url": ss.add_manga_form_chapter["URL"]
                    if ss.add_manga_form_chapter is not None
                    else "",
                }
                ss.add_manga_form_url = ""
                del ss["add_manga_chapter_options"]

            if st.form_submit_button(
                "Add Manga",
                on_click=add_manga_callback,
                use_container_width=True,
                type="primary",
            ):
                if ss.get("add_manga_manga_to_add", None) is None:
                    st.warning(
                        "Provide a manga URL and select the last read chapter first"
                    )
                else:
                    with st.spinner("Adding manga..."):
                        ss["add_manga_search_selected_manga"] = None
                        try:
                            self.api_client.add_manga(
                                ss["add_manga_manga_to_add"]["manga_url"],
                                ss["add_manga_manga_to_add"]["status"],
                                ss["add_manga_manga_to_add"]["chapter"],
                                ss["add_manga_manga_to_add"]["chapter_url"],
                            )
                        except APIException as e:
                            if (
                                "Manga added to DB, but error while adding it to Kaizoku".lower()
                                in str(e).lower()
                            ):
                                logger.exception(e)
                                ss["manga_add_warning_message"] = (
                                    "Manga added to DB, but couldn't add it to Kaizoku."
                                )
                                st.rerun()
                            else:
                                logger.exception(e)
                                ss["manga_add_error_message"] = (
                                    "Error while adding manga."
                                )
                                st.rerun()
                        else:
                            ss["manga_add_success_message"] = True
                            st.rerun()

    def show_configs(self):
        def update_configs_callback():
            self.api_client.update_dashboard_configs(
                ss.configs_select_columns_number,
                ss.configs_select_search_results_limit,
                ss.configs_select_show_background_error_warning,
            )
            ss["configs_columns_number"] = ss.configs_select_columns_number
            ss["configs_show_background_error_warning"] = (
                ss.configs_select_show_background_error_warning
            )
            ss["configs_search_results_limit"] = ss.configs_select_search_results_limit
            ss["configs_updated_success"] = True

        with st.popover(
            "Configs",
            help="Dashboard configs",
            use_container_width=True,
        ):
            with st.form(key="configs_update_configs", border=False):
                st.slider(
                    "Columns:",
                    min_value=1,
                    max_value=10,
                    value=ss["configs_columns_number"],
                    key="configs_select_columns_number",
                )

                st.slider(
                    "Search Results Limit:",
                    min_value=1,
                    max_value=50,
                    value=ss["configs_search_results_limit"],
                    key="configs_select_search_results_limit",
                )

                st.checkbox(
                    "Show background error warning",
                    value=ss["configs_show_background_error_warning"],
                    key="configs_select_show_background_error_warning",
                    help="Show a warning in the sidebar if there is a background error",
                )

                st.form_submit_button(
                    "Save",
                    type="primary",
                    on_click=update_configs_callback,
                    use_container_width=True,
                )

            if ss.get("configs_updated_success", False):
                st.success("Configs updated successfully")

    def check_dashboard_error(self):
        if ss.get("dashboard_error", False):
            st.error("An unexcepted error occurred. Please check the DASHBOARD logs.")
            st.info("You can try to refresh the page.")
            ss["dashboard_error"] = False
            st.stop()


def main(api_client):
    if (
        "configs_columns_number" not in ss
        or "configs_show_background_error_warning" not in ss
    ):
        configs = api_client.get_dashboard_configs()
        ss["configs_columns_number"] = configs["columns"]
        ss["configs_search_results_limit"] = configs["searchResultsLimit"]
        ss["configs_show_background_error_warning"] = configs[
            "showBackgroundErrorWarning"
        ]

    streamlit_general_changes = """
        <style>
            div[data-testid="stStatusWidget"] {
                display: none;
            }

            div[data-testid="stAppViewBlockContainer"] {
                padding-top: 50px !important;
            }

            div[data-testid="stSidebarUserContent"] {
                padding-top: 58px !important;
            }
        </style>
    """
    st.markdown(streamlit_general_changes, unsafe_allow_html=True)

    dashboard = MainDashboard(api_client)
    dashboard.show()


if __name__ == "__main__":
    fix_streamlit_index_html()
    api_client = get_api_client()
    api_client.check_health()

    @st.cache_data(show_spinner=False, max_entries=1, ttl=600)
    def get_manga_chapters(id: int, url: str):
        chapters = api_client.get_manga_chapters(id, url)

        return chapters

    try:
        main(api_client)
    except Exception as e:
        logger.exception(e)
        ss["dashboard_error"] = True
        st.rerun()

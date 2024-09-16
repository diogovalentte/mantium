import base64
from datetime import datetime
from io import BytesIO
from typing import Any

import streamlit as st
from PIL import Image
from src.api.api_client import get_api_client
from src.util import defaults
from src.util.add_manga import (show_add_manga_form_search,
                                show_add_manga_form_url)
from src.util.update_manga import show_update_manga
from src.util.util import (centered_container, fix_streamlit_index_html,
                           get_logger, get_relative_time, tagger)
from streamlit import session_state as ss
from streamlit_extras.stylable_container import stylable_container
from streamlit_javascript import st_javascript

st.set_page_config(
    page_title="Mantium",
    page_icon="📖",
    layout="wide",
)

logger = get_logger()


class MainDashboard:
    def __init__(self, api_client):
        self.api_client = api_client
        self.status_filter_key = 1
        self.sort_option_index = 1

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
            ss.get("mangas_sort", defaults.sort_options[self.sort_option_index]),
            ss.get("mangas_sort_reverse", False),
        )

        columns_number = ss["settings_columns_number"]
        max_number_to_show = (
            columns_number * defaults.default_number_of_rows_to_show_first
        )
        can_load_more = False
        if len(mangas) > max_number_to_show:
            self.show_mangas(st.columns(columns_number), mangas[:max_number_to_show])
            can_load_more = True
        else:
            self.show_mangas(st.columns(columns_number), mangas)

        def callback():
            ss["show_more_manga"] = True

        if not ss.get("show_more_manga", False) and can_load_more:
            st.button(
                "Show All", on_click=callback, use_container_width=True, type="primary"
            )
        if ss.get("show_more_manga", False) and can_load_more:
            self.show_mangas(st.columns(columns_number), mangas[max_number_to_show:])

        if "system_last_update_time" not in ss:
            ss["system_last_update_time"] = self.api_client.check_for_updates()

        @st.experimental_fragment(run_every=5)
        def check_for_updates():
            last_update = self.api_client.check_for_updates()
            if last_update != ss["system_last_update_time"]:
                ss["system_last_update_time"] = last_update
                st.rerun()

        check_for_updates()

        self.show_dialogs()

    def sidebar(self) -> None:
        with st.sidebar:
            self.show_background_error()

            st.text_input("Search", key="search_manga")

            def status_filter_callback():
                self.status_filter_key = ss.status_filter
                ss["show_more_manga"] = False
                js = """window.parent.document.querySelector(".main").scrollTop = 0;"""
                st_javascript(js)

            st.selectbox(
                "Filter Status",
                defaults.manga_status_options,
                index=self.sort_option_index,
                on_change=status_filter_callback,
                format_func=lambda index: defaults.manga_status_options[index],
                key="status_filter",
            )

            def sort_callback():
                self.sort_option_index = ss.mangas_sort

            st.selectbox(
                "Sort By",
                defaults.sort_options,
                index=self.status_filter_key,
                on_change=sort_callback,
                key="mangas_sort",
            )

            st.toggle("Reverse Sort", key="mangas_sort_reverse")
            st.divider()

            if not ss.get("show_more_sidebar_options", False):

                def on_click():
                    ss["show_more_sidebar_options"] = True

                st.button("More Options", on_click=on_click, use_container_width=True)
            else:
                with st.expander("More Options", expanded=True):

                    def on_search_click():
                        ss["show_add_manga_search_form"] = True

                    st.button(
                        "Add Manga by Searching",
                        type="primary",
                        use_container_width=True,
                        on_click=on_search_click,
                    )

                    def on_url_click():
                        ss["show_add_manga_url_form"] = True

                    st.button(
                        "Add Manga Using URL",
                        type="primary",
                        use_container_width=True,
                        on_click=on_url_click,
                    )

                    def on_settings_click():
                        ss["show_settings_form"] = True

                    st.button(
                        "Settings",
                        type="primary",
                        use_container_width=True,
                        on_click=on_settings_click,
                    )

                ss["show_more_sidebar_options"] = False

    def show_dialogs(self):
        """Only one dialog at the time can be shown."""
        if ss.get("show_add_manga_search_form", False):
            ss["add_manga_success_message"] = ""
            ss["add_manga_warning_message"] = ""
            ss["add_manga_error_message"] = ""
            if ss.get("add_manga_chapter_options", None) is not None:
                del ss["add_manga_chapter_options"]
            if ss.get("add_manga_search_selected_manga", None) is not None:
                del ss["add_manga_search_selected_manga"]
            ss["add_manga_search_results_mangadex"] = {}
            ss["add_manga_search_results_comick"] = {}
            ss["add_manga_search_results_mangaplus"] = {}
            ss["add_manga_search_results_mangahub"] = {}
            ss["add_manga_search_results_mangaupdates"] = {}
            ss["add_manga_search_go_back_to_tab"] = 0

            @st.experimental_dialog("Search Manga", width="large")
            def show_add_manga_form_dialog():
                show_add_manga_form_search()

            show_add_manga_form_dialog()
            ss["show_add_manga_search_form"] = False
        elif ss.get("show_add_manga_url_form", False):
            ss["add_manga_success_message"] = ""
            ss["add_manga_warning_message"] = ""
            ss["add_manga_error_message"] = ""
            if ss.get("add_manga_chapter_options", None) is not None:
                del ss["add_manga_chapter_options"]

            @st.experimental_dialog("Add Manga Using URL")
            def show_add_manga_form_dialog():
                show_add_manga_form_url()

            show_add_manga_form_dialog()
            ss["show_add_manga_url_form"] = False
        elif ss.get("show_settings_form", False):
            ss["configs_update_success_message"] = ""
            ss["configs_update_warning_message"] = ""
            ss["configs_update_error_message"] = ""

            @st.experimental_dialog("Settings")
            def show_settings_dialog():
                self.show_settings()

            show_settings_dialog()
            ss["show_settings_form"] = False
        elif ss.get("highlighted_manga", None) is not None:
            manga = ss["highlighted_manga"]

            @st.experimental_dialog(manga["Name"])
            def show_highlighted_manga_dialog():
                show_update_manga(manga)

            show_highlighted_manga_dialog()
            ss["highlighted_manga"] = None
        elif ss.get("update_manga_success_message", "") != "":

            @st.experimental_dialog("Manga Updated")
            def show_add_manga_message():
                if ss.get("update_manga_success_message", "") != "":
                    st.success(ss["update_manga_success_message"])

            show_add_manga_message()

            ss["update_manga_success_message"] = ""
        elif (
            ss.get("add_manga_success_message", "") != ""
            or ss.get("add_manga_warning_message", "") != ""
        ):

            @st.experimental_dialog("Add Manga")
            def show_add_manga_message():
                if ss.get("add_manga_success_message", "") != "":
                    st.success(ss["add_manga_success_message"])
                if ss.get("add_manga_warning_message", "") != "":
                    st.warning(ss["add_manga_warning_message"])

            show_add_manga_message()

            ss["add_manga_success_message"] = ""
            ss["add_manga_warning_message"] = ""
        elif (
            ss.get("configs_update_success_message", "") != ""
            or ss.get("configs_update_warning_message", "") != ""
        ):

            @st.experimental_dialog("Settings")
            def show_settings_message():
                if ss.get("configs_update_success_message", "") != "":
                    st.success(ss["configs_update_success_message"])
                if ss.get("configs_update_warning_message", "") != "":
                    st.warning(ss["configs_update_warning_message"])

            show_settings_message()

            ss["configs_update_success_message"] = ""
            ss["configs_update_warning_message"] = ""

    def show_background_error(self):
        if ss["settings_show_background_error_warning"]:
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
                        with stylable_container(
                            key="delete_background_error_button",
                            css_styles="""
                                button {
                                    background-color: red;
                                    color: white;
                                }
                            """,
                        ):
                            if st.button(
                                "Delete Error",
                                use_container_width=True,
                                help="Delete the last background error",
                                on_click=self.api_client.delete_last_background_error,
                                key="delete_background_error_button_from_dialog",
                            ):
                                st.rerun()

                    if st.button(
                        "See error",
                        type="primary",
                        help="See error message",
                        use_container_width=True,
                    ):
                        show_error_message_dialog()
                    with stylable_container(
                        key="delete_background_error_button",
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
                            key="delete_background_error_button_from_expander",
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
                        self.show_manga(manga)
            col_index += 1

    def show_manga(self, manga: dict[str, Any]):
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
                f'**Status**: <span style="float: right;">{defaults.manga_status_options[manga["Status"]]}</span>',
                unsafe_allow_html=True,
            )

        chapter_tag_content = f"""
            <a href="{{}}" target="_blank" style="text-decoration: none; color: {defaults.chapter_link_tag_text_color}">
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
            defaults.chapter_link_tag_background_color,
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
            defaults.chapter_link_tag_background_color,
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
                        manga["ID"], manga["URL"], manga["InternalID"], ""
                    )

            st.button(
                "Set last read",
                use_container_width=True,
                on_click=set_last_read,
                key=f"set_last_read_{manga['ID']}",
                disabled=not unread,
            )
        with c2:

            def highlight_manga():
                ss["highlighted_manga"] = manga

            st.button(
                "Highlight",
                use_container_width=True,
                type="primary",
                key=f"highlight_{manga['ID']}",
                on_click=highlight_manga,
            )

    def show_settings(self):
        with st.form(key="configs_update_configs", border=False):
            st.slider(
                "Columns:",
                min_value=1,
                max_value=10,
                value=ss["settings_columns_number"],
                key="configs_select_columns_number",
            )

            st.slider(
                "Search Results Limit:",
                min_value=1,
                max_value=50,
                value=ss["settings_search_results_limit"],
                help="The maximum number of search results to show when searching for a manga to add to the dashboard. It doesn't work very well with MangaUpdates.",
                key="configs_select_search_results_limit",
            )

            st.checkbox(
                "Show background error warning",
                value=ss["settings_show_background_error_warning"],
                key="configs_select_show_background_error_warning",
                help="Show a warning in the sidebar if there is a background error",
            )

            if st.form_submit_button(
                "Save",
                type="primary",
                use_container_width=True,
            ):
                try:
                    self.api_client.update_dashboard_configs(
                        ss.configs_select_columns_number,
                        ss.configs_select_search_results_limit,
                        ss.configs_select_show_background_error_warning,
                    )
                    ss["settings_columns_number"] = ss.configs_select_columns_number
                    ss["settings_show_background_error_warning"] = (
                        ss.configs_select_show_background_error_warning
                    )
                    ss["settings_search_results_limit"] = (
                        ss.configs_select_search_results_limit
                    )
                    ss["configs_update_success_message"] = (
                        "Settings saved successfully."
                    )
                    st.rerun()
                except Exception as e:
                    logger.exception(e)
                    ss["configs_update_error_message"] = "Error while saving settings."

        if ss.get("configs_update_error_message", "") != "":
            st.error(ss["configs_update_error_message"])

    def check_dashboard_error(self):
        if ss.get("dashboard_error", False):
            st.error("An unexcepted error occurred. Please check the DASHBOARD logs.")
            st.info("You can try to refresh the page.")
            ss["dashboard_error"] = False
            st.stop()


def main(api_client):
    if (
        "settings_columns_number" not in ss
        or "settings_show_background_error_warning" not in ss
    ):
        configs = api_client.get_dashboard_configs()
        ss["settings_columns_number"] = configs["columns"]
        ss["settings_search_results_limit"] = configs["searchResultsLimit"]
        ss["settings_show_background_error_warning"] = configs[
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

    try:
        main(api_client)
    except Exception as e:
        logger.exception(e)
        ss["dashboard_error"] = True
        st.rerun()

import base64
from datetime import datetime
from io import BytesIO
from typing import Any

import streamlit as st
from browser_detection import browser_detection_engine
from PIL import Image
from src.api.api_client import get_api_client
from src.util import defaults
from src.util.add_manga import show_add_manga_form
from src.util.update_manga import (
    show_update_multimanga_form,
    show_update_multimanga_mangas_form,
)
from src.util.util import centered_container, get_logger, get_relative_time, tagger
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
        self.set_css()

        ss["is_dialog_open"] = False
        self.check_dashboard_error()

        self.sidebar()

        mangas = self.api_client.get_mangas()
        filter_by_status = ss.get(
            "status_filter",
            self.status_filter_key,
        )
        filter_by_name_term = ss.get("search_manga", "").upper()
        if filter_by_status != 0 and filter_by_name_term != "":
            mangas = [
                manga
                for manga in mangas
                if manga["Status"] == filter_by_status
                and filter_by_name_term in ("".join(manga["SearchNames"])).upper()
            ]
        elif filter_by_status != 0:
            mangas = [manga for manga in mangas if manga["Status"] == filter_by_status]
        elif filter_by_name_term != "":
            mangas = [
                manga
                for manga in mangas
                if filter_by_name_term in ("".join(manga["SearchNames"])).upper()
            ]

        mangas = self.api_client.sort_mangas(
            mangas,
            ss.get("mangas_sort", defaults.sort_options[self.sort_option_index]),
            ss.get("mangas_sort_reverse", False),
        )

        def callback():
            ss["show_more_manga"] = True

        can_load_more = False

        if ss["settings_display_mode"] == "List View":
            max_mangas_to_show = defaults.list_view_number_of_rows_to_show_first
            if len(mangas) > max_mangas_to_show:
                self.show_mangas_list_view(mangas[:max_mangas_to_show])
                can_load_more = True
            else:
                self.show_mangas_list_view(mangas)

            if not ss.get("show_more_manga", False) and can_load_more:
                st.button(
                    "Show All",
                    on_click=callback,
                    use_container_width=True,
                    type="primary",
                )
            if ss.get("show_more_manga", False) and can_load_more:
                self.show_mangas_list_view(mangas[max_mangas_to_show:])
        else:
            if not ss["is_mobile"]:
                columns_number = ss["settings_columns_number"]
                max_mangas_to_show = (
                    columns_number * defaults.grid_view_number_of_rows_to_show_first
                )
            else:
                columns_number = 1
                max_mangas_to_show = (
                    ss["settings_columns_number"]
                    * defaults.grid_view_number_of_rows_to_show_first
                )
            if len(mangas) > max_mangas_to_show:
                self.show_mangas_grid_view(
                    st.columns(columns_number), mangas[:max_mangas_to_show]
                )
                can_load_more = True
            else:
                self.show_mangas_grid_view(st.columns(columns_number), mangas)

            if not ss.get("show_more_manga", False) and can_load_more:
                st.button(
                    "Show All",
                    on_click=callback,
                    use_container_width=True,
                    type="primary",
                )
            if ss.get("show_more_manga", False) and can_load_more:
                self.show_mangas_grid_view(
                    st.columns(columns_number), mangas[max_mangas_to_show:]
                )

        if "system_last_update_time" not in ss:
            ss["system_last_update_time"] = self.api_client.check_for_updates()

        self.update_dashboard_job()

        self.show_forms()

    def set_css(self):
        improve_css = """
            <style>
                /* Hide the header link button */
                button[title="View fullscreen"]{
                    display: none !important;
                }

                /* Hide the browser detection engine div */
                div.st-key-browser_engine {
                    display: none !important;
                }
            </style>

            <style>
                /* Hide the header link button */
                h1.manga_header > span {
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
                    text-align: center;
                    margin-top: 0px;
                    margin-bottom: 0px;
                    font-size: 30px;
                }

                a.manga_header {
                    text-decoration: none;
                    color: inherit;
                }
                a.manga_header:hover {
                    color: #04c9b7;
                }
                span.manga_header:hover {
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

            <style>
                /* Center images in streamlit>1.39, but it's kind slow
                div[data-testid="stElementContainer"]:has(div[data-testid="stImageContainer"]) {
                    display: grid;
                    align-items: center;
                }
                */
            </style>

            <style>
                /* General changes */
                div[data-testid="stStatusWidget"] {
                    display: none;
                }

                div[data-testid="stMainBlockContainer"] {
                    padding-top: 50px !important;
                }
            </style>
        """
        st.markdown(improve_css, unsafe_allow_html=True)

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
                "Status",
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

            with st.expander("More Options"):

                def on_add_manga_click(form_type: str):
                    ss["show_add_manga_form"] = form_type

                st.button(
                    "Add Manga by Searching",
                    type="primary",
                    use_container_width=True,
                    on_click=on_add_manga_click,
                    args=("search",),
                )

                st.button(
                    "Add Manga Using URL",
                    type="primary",
                    use_container_width=True,
                    on_click=on_add_manga_click,
                    args=("url",),
                )

                st.button(
                    "Add Custom Manga",
                    type="primary",
                    use_container_width=True,
                    on_click=on_add_manga_click,
                    args=("custom",),
                )

                def on_settings_click():
                    ss["show_settings_form"] = True

                st.button(
                    "Settings",
                    type="primary",
                    use_container_width=True,
                    on_click=on_settings_click,
                )

    def show_forms(self):
        if ss.get("show_add_manga_form", "") != "":
            show_add_manga_form(ss["show_add_manga_form"])
            ss["show_add_manga_form"] = ""
        elif ss.get("highlighted_manga", None) is not None:
            show_update_multimanga_form(ss["highlighted_manga"])
            ss["highlighted_manga"] = None
        elif ss.get("highlighted_multimanga", None) is not None:
            show_update_multimanga_mangas_form(ss["highlighted_multimanga"])
            ss["highlighted_multimanga"] = None
        elif ss.get("update_manga_success_message", "") != "":

            @st.dialog("Update Manga")
            def show():
                ss["is_dialog_open"] = True
                st.success(ss["update_manga_success_message"])
                ss["update_manga_success_message"] = ""

            show()
        elif ss.get("add_manga_success_message", "") != "":

            @st.dialog("Add Manga")
            def show():
                ss["is_dialog_open"] = True
                st.success(ss["add_manga_success_message"])
                ss["add_manga_success_message"] = ""

            show()
        elif ss.get("add_manga_warning_message", "") != "":

            @st.dialog("Add Manga")
            def show():
                ss["is_dialog_open"] = True
                st.warning(ss["add_manga_warning_message"])
                ss["add_manga_warning_message"] = ""

            show()
        elif ss.get("show_settings_form", False):
            self.show_settings()
            ss["show_settings_form"] = False
        elif (
            ss.get("configs_update_error_message", "") != ""
            or ss.get("configs_update_success_message", "") != ""
        ):

            @st.dialog("Settings")
            def show_configs_update_message():
                ss["is_dialog_open"] = True
                if ss.get("configs_update_error_message", "") != "":
                    st.error(ss["configs_update_error_message"])
                else:
                    st.success(ss["configs_update_success_message"])
                ss["configs_update_error_message"] = ""
                ss["configs_update_success_message"] = ""

            show_configs_update_message()

    def show_background_error(self):
        if ss["settings_show_background_error_warning"]:
            last_background_error = self.api_client.get_last_background_error()
            if len(last_background_error["message"]) > 3000:
                last_background_error["message"] = (
                    last_background_error["message"][:3000] + "..."
                )
            if last_background_error["message"] != "":
                with st.expander("An error occurred in the background!", expanded=True):
                    logger.error(
                        f"Background error: {last_background_error['message']}"
                    )
                    st.info(f"Time: {last_background_error['time']}")

                    @st.dialog("Last Background Error Message", width="large")
                    def show_error_message_dialog():
                        ss["is_dialog_open"] = True
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

    def show_mangas_grid_view(self, cols_list: list, mangas: list[dict[str, Any]]):
        """Show mangas in the cols_list columns in grid view.

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
                        self.show_manga_original(manga)
            col_index += 1

    def show_manga_original(self, manga: dict[str, Any]):
        unread = (
            manga["LastReadChapter"]["Chapter"]
            != manga["LastReleasedChapter"]["Chapter"]
        )

        st.markdown(
            f"""<h1
                class="manga_header" style='margin-top: 16px; margin-bottom: 8px; {"animation: pulse 2s infinite alternate;" if unread else ""}'>
                    <div style='position: relative; display: flex; box-sizing: border-box;'>
                        <span>
                            {'<a class="manga_header" href="{}" target="_blank">{}</a>'.format(manga["URL"], manga["Name"]) if manga["URL"] != "" else f'<span class="manga_header">{manga["Name"]}</span>'}
                        </span>
                    </div>
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
                f"""<img src="{manga["CoverImgURL"]}" width="250" height="355" style="margin-bottom: 16px;"/>""",
                unsafe_allow_html=True,
            )
        else:
            st.markdown(
                f"""<img src="{defaults.DEFAULT_MANGA_COVER}" width="250" height="355" style="margin-bottom: 16px;"/>""",
                unsafe_allow_html=True,
            )

        if ss.get("status_filter", 1) == 0:
            st.write(
                f'**Status**: <span style="float: right;">{defaults.manga_status_options[manga["Status"]]}</span>',
                unsafe_allow_html=True,
            )

        def highlight_manga():
            ss["highlighted_manga"] = manga

        if manga["Source"] != defaults.CUSTOM_MANGA_SOURCE:
            chapter_tag_content = f"""
                <a href="{{}}" target="_blank" style="text-decoration: none; color: {defaults.chapter_link_tag_text_color}">
                    <span>{{}}</span>
                </a>
            """

            chapter = chapter_tag_content.format(
                manga["LastReleasedChapter"]["URL"],
                (
                    f'Ch. {manga["LastReleasedChapter"]["Chapter"]}'
                    if manga["LastReleasedChapter"]["Chapter"] != ""
                    else "N/A"
                ),
            )
            release_date = (
                manga["LastReleasedChapter"]["UpdatedAt"]
                if manga["LastReleasedChapter"]["UpdatedAt"] != datetime.min
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
                (
                    f'Ch. {manga["LastReadChapter"]["Chapter"]}'
                    if manga["LastReadChapter"]["Chapter"] != ""
                    else "N/A"
                ),
            )
            read_date = (
                manga["LastReadChapter"]["UpdatedAt"]
                if manga["LastReadChapter"]["UpdatedAt"] != datetime.min
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
                f"""**Read Date**: <span style="float: right;" title="{read_date}">{relative_read_date}</span>""",
                unsafe_allow_html=True,
            )

            c1, c2 = st.columns(2)
            with c1:

                def set_last_read():
                    if (
                        manga.get("LastReadChapter", {}).get("Chapter")
                        != manga["LastReleasedChapter"]["Chapter"]
                    ):
                        if manga["MultiMangaID"] == 0:
                            self.api_client.update_manga_last_read_chapter(
                                manga["ID"], manga["URL"], manga["InternalID"]
                            )
                        else:
                            self.api_client.update_multimanga_last_read_chapter(
                                manga["MultiMangaID"], manga["ID"]
                            )

                st.button(
                    "Set last read",
                    use_container_width=True,
                    on_click=set_last_read,
                    key=f"set_last_read_{manga['ID']}",
                    disabled=not unread,
                )
            with c2:
                st.button(
                    "Highlight",
                    use_container_width=True,
                    type="primary",
                    key=f"highlight_{manga['ID']}",
                    on_click=highlight_manga,
                )
        else:
            if manga["LastReadChapter"]["URL"] != "":
                chapter_tag_content = f"""
                    <a href="{manga["LastReadChapter"]["URL"]}" target="_blank" style="text-decoration: none; color: {defaults.chapter_link_tag_text_color}">
                        <span>{"Ch. {}".format(manga["LastReadChapter"]["Chapter"]) if manga["LastReadChapter"]["Chapter"] != "" else "N/A"}</span>
                    </a>
                """
            else:
                chapter_tag_content = f"""
                    <span style="color: {defaults.chapter_link_tag_text_color}">{"Ch. {}".format(manga["LastReadChapter"]["Chapter"]) if manga["LastReadChapter"]["Chapter"] != "" else "N/A"}</span>
                """

            read_date = (
                manga["LastReadChapter"]["UpdatedAt"]
                if manga["LastReadChapter"]["UpdatedAt"] != datetime.min
                else "N/A"
            )
            if read_date != "N/A":
                relative_read_date = get_relative_time(read_date)
            else:
                relative_read_date = read_date

            tagger(
                f"<strong>{'Next' if manga['LastReadChapter']['Chapter'] != manga['LastReleasedChapter']['Chapter'] else 'Last Read'} Chapter:</strong>",
                chapter_tag_content,
                defaults.chapter_link_tag_background_color,
                "float: right;",
            )
            st.caption(
                f"**Updated Date**: <span style='float: right;' title='{read_date}'>{relative_read_date}</span>",
                unsafe_allow_html=True,
            )

            st.markdown("""<div style="height: 24px"></div>""", unsafe_allow_html=True)

            def set_no_more_chapters():
                api_client.update_custom_manga_has_more_chapters(False, manga["ID"], "")

            st.button(
                "No more chapters",
                use_container_width=True,
                on_click=set_no_more_chapters,
                key=f"set_no_more_chapters_{manga['ID']}",
                disabled=not unread,
            )

            st.button(
                "Highlight",
                use_container_width=True,
                type="primary",
                key=f"highlight_{manga['ID']}",
                on_click=highlight_manga,
            )

    def show_mangas_list_view(self, mangas: list[dict[str, Any]]):
        """Show mangas in the list view.

        Args:
            mangas (dict): A list of mangas.
        """
        for manga in mangas:
            with st.container(border=True):
                self.show_manga_list_view(manga)

    def show_manga_list_view(self, manga: dict[str, Any]):
        is_custom_manga = manga["Source"] == defaults.CUSTOM_MANGA_SOURCE
        show_status = ss.get("status_filter", 1) == 0

        if not is_custom_manga:
            if not show_status:
                (
                    cover_col,
                    name_col,
                    last_released_chap_col,
                    last_released_chap_date_col,
                    last_read_chap_col,
                    last_read_chap_date_col,
                    set_last_read_col,
                    highlight_col,
                ) = st.columns(
                    [9, 40, 22, 12, 22, 12, 20, 20],
                    gap="small",
                    vertical_alignment="center",
                )
            else:
                (
                    cover_col,
                    name_col,
                    status_col,
                    last_released_chap_col,
                    last_released_chap_date_col,
                    last_read_chap_col,
                    last_read_chap_date_col,
                    set_last_read_col,
                    highlight_col,
                ) = st.columns(
                    [9, 28, 12, 22, 12, 22, 12, 20, 20],
                    gap="small",
                    vertical_alignment="center",
                )
        else:
            if not show_status:
                (
                    cover_col,
                    name_col,
                    last_read_chap_col,
                    last_read_chap_date_col,
                    set_last_read_col,
                    highlight_col,
                ) = st.columns(
                    [9, 74, 22, 12, 20, 20],
                    gap="small",
                    vertical_alignment="center",
                )
            else:
                (
                    cover_col,
                    name_col,
                    status_col,
                    last_read_chap_col,
                    last_read_chap_date_col,
                    set_last_read_col,
                    highlight_col,
                ) = st.columns(
                    [9, 62, 12, 22, 12, 20, 20],
                    gap="small",
                    vertical_alignment="center",
                )

        unread = (
            manga["LastReadChapter"]["Chapter"]
            != manga["LastReleasedChapter"]["Chapter"]
        )

        with name_col:
            st.markdown(
                f"""<h1
                    class="manga_header" style='font-size: 25px; {"animation: pulse 2s infinite alternate;" if unread else ""}'>
                        <div style='position: relative; display: flex; box-sizing: border-box;'>
                            <span>
                                {'<a class="manga_header" href="{}" target="_blank">{}</a>'.format(manga["URL"], manga["Name"]) if manga["URL"] != "" else f'<span class="manga_header">{manga["Name"]}</span>'}
                            </span>
                        </div>
                    </h1>
                """,
                unsafe_allow_html=True,
            )

        with cover_col:
            if manga["CoverImg"] is not None:
                img_bytes = base64.b64decode(manga["CoverImg"])
                img = BytesIO(img_bytes)
                if True:
                    img = Image.open(img)
                    img = img.resize((52, 75))
                st.image(img)
            elif manga["CoverImgURL"] != "":
                st.markdown(
                    f"""<img src="{manga["CoverImgURL"]}" width="52" height="75"/>""",
                    unsafe_allow_html=True,
                )
            else:
                st.markdown(
                    f"""<img src="{defaults.DEFAULT_MANGA_COVER}" width="52" height="75"/>""",
                    unsafe_allow_html=True,
                )

        if show_status:
            with status_col:
                st.write("**Status**:", unsafe_allow_html=True)
                st.write(
                    f'<span style="color: #d6d6d9;">{defaults.manga_status_options[manga["Status"]]}</span>',
                    unsafe_allow_html=True,
                )

        def highlight_manga():
            ss["highlighted_manga"] = manga

        if not is_custom_manga:
            chapter_tag_content = f"""
                <a href="{{}}" target="_blank" style="text-decoration: none; color: {defaults.chapter_link_tag_text_color}">
                    <span>{{}}</span>
                </a>
            """

            chapter = chapter_tag_content.format(
                manga["LastReleasedChapter"]["URL"],
                (
                    f'Ch. {manga["LastReleasedChapter"]["Chapter"]}'
                    if manga["LastReleasedChapter"]["Chapter"] != ""
                    else "N/A"
                ),
            )
            release_date = (
                manga["LastReleasedChapter"]["UpdatedAt"]
                if manga["LastReleasedChapter"]["UpdatedAt"] != datetime.min
                else "N/A"
            )
            if release_date != "N/A":
                relative_release_date = get_relative_time(release_date)
            else:
                relative_release_date = release_date

            with last_released_chap_col:
                st.write(
                    "<strong>Last Released Chapter:</strong>", unsafe_allow_html=True
                )
                tagger(
                    "",
                    chapter,
                    defaults.chapter_link_tag_background_color,
                )
            with last_released_chap_date_col:
                st.caption("Release Date:")
                st.caption(
                    f'<span style="color: #d6d6d9" title="{release_date}">{relative_release_date}</span>',
                    unsafe_allow_html=True,
                )

            chapter = chapter_tag_content.format(
                manga["LastReadChapter"]["URL"],
                (
                    f'Ch. {manga["LastReadChapter"]["Chapter"]}'
                    if manga["LastReadChapter"]["Chapter"] != ""
                    else "N/A"
                ),
            )
            read_date = (
                manga["LastReadChapter"]["UpdatedAt"]
                if manga["LastReadChapter"]["UpdatedAt"] != datetime.min
                else "N/A"
            )
            if read_date != "N/A":
                relative_read_date = get_relative_time(read_date)
            else:
                relative_read_date = read_date

            with last_read_chap_col:
                st.write("<strong>Last Read Chapter:</strong>", unsafe_allow_html=True)
                tagger(
                    "",
                    chapter,
                    defaults.chapter_link_tag_background_color,
                )
            with last_read_chap_date_col:
                st.caption("Read Date:")
                st.caption(
                    f'<span style="color: #d6d6d9" title="{read_date}">{relative_read_date}</span>',
                    unsafe_allow_html=True,
                )

            with set_last_read_col:

                def set_last_read():
                    if (
                        manga.get("LastReadChapter", {}).get("Chapter")
                        != manga["LastReleasedChapter"]["Chapter"]
                    ):
                        if manga["MultiMangaID"] == 0:
                            self.api_client.update_manga_last_read_chapter(
                                manga["ID"], manga["URL"], manga["InternalID"]
                            )
                        else:
                            self.api_client.update_multimanga_last_read_chapter(
                                manga["MultiMangaID"], manga["ID"]
                            )

                st.button(
                    "Set last read",
                    use_container_width=True,
                    on_click=set_last_read,
                    key=f"set_last_read_{manga['ID']}",
                    disabled=not unread,
                )
            with highlight_col:
                st.button(
                    "Highlight",
                    use_container_width=True,
                    type="primary",
                    key=f"highlight_{manga['ID']}",
                    on_click=highlight_manga,
                )
        else:
            if manga["LastReadChapter"]["URL"] != "":
                chapter_tag_content = f"""
                    <a href="{manga["LastReadChapter"]["URL"]}" target="_blank" style="text-decoration: none; color: {defaults.chapter_link_tag_text_color}">
                        <span>{"Ch. {}".format(manga["LastReadChapter"]["Chapter"]) if manga["LastReadChapter"]["Chapter"] != "" else "N/A"}</span>
                    </a>
                """
            else:
                chapter_tag_content = f"""
                    <span style="color: {defaults.chapter_link_tag_text_color}">{"Ch. {}".format(manga["LastReadChapter"]["Chapter"]) if manga["LastReadChapter"]["Chapter"] != "" else "N/A"}</span>
                """

            read_date = (
                manga["LastReadChapter"]["UpdatedAt"]
                if manga["LastReadChapter"]["UpdatedAt"] != datetime.min
                else "N/A"
            )
            if read_date != "N/A":
                relative_read_date = get_relative_time(read_date)
            else:
                relative_read_date = read_date

            with last_read_chap_col:
                st.write(
                    f"<strong>{'Next' if manga['LastReadChapter']['Chapter'] != manga['LastReleasedChapter']['Chapter'] else 'Last Read'} Chapter:</strong>",
                    unsafe_allow_html=True,
                )
                tagger(
                    "",
                    chapter_tag_content,
                    defaults.chapter_link_tag_background_color,
                )
            with last_read_chap_date_col:
                st.caption("Updated Date:")
                st.caption(
                    f"<span style='color: #d6d6d9;' title='{read_date}'>{relative_read_date}</span>",
                    unsafe_allow_html=True,
                )

            def set_no_more_chapters():
                api_client.update_custom_manga_has_more_chapters(False, manga["ID"], "")

            with set_last_read_col:
                st.button(
                    "No more chapters",
                    use_container_width=True,
                    on_click=set_no_more_chapters,
                    key=f"set_no_more_chapters_{manga['ID']}",
                    disabled=not unread,
                )

            with highlight_col:
                st.button(
                    "Highlight",
                    use_container_width=True,
                    type="primary",
                    key=f"highlight_{manga['ID']}",
                    on_click=highlight_manga,
                )

    @st.dialog("Settings")
    def show_settings(self):
        ss["is_dialog_open"] = True
        with st.form(key="configs_update_configs", border=False):
            with st.expander("Display"):
                st.selectbox(
                    (
                        "Display Mode"
                        if not ss["is_mobile"]
                        else "Display Mode (only Grid View in mobile)"
                    ),
                    defaults.display_modes,
                    index=(
                        defaults.display_modes.index(ss["settings_display_mode"])
                        if ss["settings_display_mode"] in defaults.display_modes
                        else 0
                    ),
                    disabled=ss["is_mobile"],
                    help="Select the dashboard display mode",
                    key="configs_select_display_mode",
                )

                if ss["is_mobile"]:
                    columns_settings_label = "Columns (not available in mobile):"
                elif ss["settings_display_mode"] == "Grid View":
                    columns_settings_label = "Columns:"
                else:
                    columns_settings_label = "Columns (available in Grid View only):"

                st.slider(
                    columns_settings_label,
                    min_value=defaults.columns_min_value,
                    max_value=defaults.columns_max_value,
                    value=ss["settings_columns_number"],
                    disabled=(ss["settings_display_mode"] == "List View")
                    or ss["is_mobile"],
                    key="configs_select_columns_number",
                )

                st.slider(
                    "Search Results Limit:",
                    min_value=defaults.search_results_limit_min_value,
                    max_value=defaults.search_results_limit_max_value,
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

            with st.expander("Integrations"):
                st.checkbox(
                    "Add multimanga mangas to download integrations",
                    value=ss[
                        "settings_add_all_multimanga_mangas_to_download_integrations"
                    ],
                    help="By default, add only the multimangas' first manga to download integrations. If checked, when adding additional mangas to a multimanga, add them to download integrations too.",
                    key="configs_add_all_multimanga_mangas_to_download_integrations",
                )
                st.checkbox(
                    "Enqueue chapters to download when adding manga to Suwayomi",
                    value=ss["settings_enqueue_all_suwayomi_chapters_to_download"],
                    help="By default, enqueue to download all chapters from the mangas that are added to the Suwayomi integration (expect ComicK mangas).",
                    key="configs_enqueue_all_suwayomi_chapters_to_download",
                )

            if st.form_submit_button(
                "Save",
                type="primary",
                use_container_width=True,
            ):
                try:
                    new_configs = {
                        "dashboard": {
                            "columns": ss.configs_select_columns_number,
                            "showBackgroundErrorWarning": ss.configs_select_show_background_error_warning,
                            "searchResultsLimit": ss.configs_select_search_results_limit,
                            "displayMode": ss.configs_select_display_mode,
                        },
                        "integrations": {
                            "addAllMultiMangaMangasToDownloadIntegrations": ss.configs_add_all_multimanga_mangas_to_download_integrations,
                            "enqueueAllSuwayomiChaptersToDownload": ss.configs_enqueue_all_suwayomi_chapters_to_download,
                        },
                    }
                    self.api_client.update_dashboard_configs(new_configs)

                    ss["settings_columns_number"] = ss.configs_select_columns_number
                    ss[
                        "settings_show_background_error_warning"
                    ] = ss.configs_select_show_background_error_warning
                    ss["settings_display_mode"] = ss.configs_select_display_mode
                    ss[
                        "settings_search_results_limit"
                    ] = ss.configs_select_search_results_limit
                    ss[
                        "settings_add_all_multimanga_mangas_to_download_integrations"
                    ] = ss.configs_add_all_multimanga_mangas_to_download_integrations
                    ss[
                        "settings_enqueue_all_suwayomi_chapters_to_download"
                    ] = ss.configs_enqueue_all_suwayomi_chapters_to_download

                    ss["configs_update_success_message"] = "Settings saved successfully"
                    st.rerun()
                except Exception as e:
                    logger.exception(e)
                    ss["configs_update_error_message"] = "Error while saving settings"
                    st.rerun()

    def check_dashboard_error(self):
        if ss.get("dashboard_error", False):
            st.error("An unexcepted error occurred. Please check the DASHBOARD logs.")
            st.info("You can try to refresh the page.")
            ss["dashboard_error"] = False
            st.stop()

    @st.fragment(run_every=5)
    def update_dashboard_job(self):
        last_update = self.api_client.check_for_updates()
        # ss["is_dialog_open"] is used to prevent the dialog from closing when the user is interacting with it
        # It's not the perfect solution, but it's the best I could come up with.
        # It's reseted to True when the app reruns
        if last_update != ss["system_last_update_time"] and not ss["is_dialog_open"]:
            ss["system_last_update_time"] = last_update
            st.rerun()


def main(api_client):
    ss["is_mobile"] = browser_detection_engine()["isMobile"]

    if "settings_columns_number" not in ss:
        configs = api_client.get_dashboard_configs()
        dashboard = configs["dashboard"]
        integrations = configs["integrations"]
        ss["settings_display_mode"] = (
            dashboard["displayMode"]
            if not ss["is_mobile"]
            else defaults.display_modes[0]
        )
        ss["settings_columns_number"] = dashboard["columns"]
        ss["settings_search_results_limit"] = dashboard["searchResultsLimit"]
        ss["settings_show_background_error_warning"] = dashboard[
            "showBackgroundErrorWarning"
        ]
        ss[
            "settings_add_all_multimanga_mangas_to_download_integrations"
        ] = integrations["addAllMultiMangaMangasToDownloadIntegrations"]
        ss["settings_enqueue_all_suwayomi_chapters_to_download"] = integrations[
            "enqueueAllSuwayomiChaptersToDownload"
        ]

    dashboard = MainDashboard(api_client)
    dashboard.show()


if __name__ == "__main__":
    api_client = get_api_client()
    api_client.check_health()

    try:
        main(api_client)
    except Exception as e:
        logger.exception(e)
        ss["dashboard_error"] = True
        st.rerun()

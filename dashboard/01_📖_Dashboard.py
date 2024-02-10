import base64
import logging
from io import BytesIO
from typing import Any

import streamlit as st
from PIL import Image
from src.api.api_client import get_api_client
from src.exceptions import APIException
from src.util import centered_container, get_relative_time, tagger
from streamlit import session_state as ss
from streamlit_extras.stylable_container import stylable_container

st.set_page_config(
    page_title="Mantium",
    page_icon="📖",
    layout="wide",
)

st.write("#")  # Without this, the app loads in the middle of the page


class MainDashboard:
    def __init__(self):
        self.api_client = get_api_client()
        self.manga_status_options = {
            1: "📖 Reading",
            2: "✅ Completed",
            3: "🚧 On Hold",
            4: "❌ Dropped",
            5: "📅 Plan to Read",
        }
        self.default_status_filter = 1
        self.sort_options = [
            "Name",
            "Unread",
            "Last Read",
            "Chapters Released",
            "Released Chapter Date",
        ]
        self.default_sort_option_index = 1
        self.default_chapter_link_background_color = "rgb(219 233 254)"
        self.default_chapter_link_text_color = "rgb(59 130 246)"

    def show(self):
        mangas = self.api_client.get_mangas()
        mangas = [
            manga
            for manga in mangas
            if ss.get(
                "status_filter",
                self.manga_status_options[self.default_status_filter],
            )
            == "📚 All"
            or self.manga_status_options[manga["Status"]]
            == ss.get(
                "status_filter",
                self.manga_status_options[self.default_status_filter],
            )
        ]
        mangas = [
            manga
            for manga in mangas
            if ss.get("search_manga", "").upper() in manga["Name"].upper()
        ]

        mangas = self.api_client.sort_mangas(
            mangas,
            ss.get("mangas_sort", self.sort_options[self.default_sort_option_index]),
            ss.get("mangas_sort_reverse", False),
        )
        self.show_mangas(st.columns(6), mangas)

        self.sidebar()

    def sidebar(self) -> dict[str, Any]:
        with st.sidebar:
            st.text_input("Search", key="search_manga")

            return_dict = dict()
            # filters and sort options
            filter_options = list(self.manga_status_options.values())
            filter_options.insert(0, "📚 All")

            def status_filter_callback():
                self.default_status_filter = ss.status_filter

            return_dict["status_filter"] = st.selectbox(
                "Filter Status",
                filter_options,
                index=self.default_sort_option_index,
                on_change=status_filter_callback,
                key="status_filter",
            )

            def sort_callback():
                self.default_sort_option_index = ss.mangas_sort

            return_dict["sort_option"] = st.selectbox(
                "Sort By",
                self.sort_options,
                index=self.default_status_filter,
                on_change=sort_callback,
                key="mangas_sort",
            )

            return_dict["reverse_sort"] = st.toggle(
                "Reverse Sort", key="mangas_sort_reverse"
            )
            st.divider()

            highlight_manga_container = st.empty()

            with st.expander("Add Manga"):
                self.show_add_manga_form()

            manga_to_highlight = ss.get("manga_to_highlight", None)
            if manga_to_highlight is not None:
                with highlight_manga_container:
                    with st.container(border=True):
                        self.show_highlighted_manga(manga_to_highlight)

        return return_dict

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
                with st.container(border=True, height=763):
                    with centered_container("center_container"):
                        self.show_manga(manga)
            col_index += 1

    def show_manga(self, manga: dict[str, Any]):
        unread = (
            manga["LastReadChapter"]["Chapter"] != manga["LastUploadChapter"]["Chapter"]
        )

        # Try to make the title fit in the container the best way
        # Also try to make the containers the same size
        default_size = 36
        characters = len(manga["Name"])
        if characters < 15:
            font_size = default_size
            margin = 0
        elif characters < 30:
            font_size = 20
            margin = (default_size - font_size) / 2 + 1.6
        else:
            font_size = 15
            margin = (default_size - font_size) / 2 + 1.6
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
            st.image(manga["CoverImgURL"])
        # Hide the "View fullscreen" button from the image
        hide_img_fs = """
        <style>
            button[title="View fullscreen"]{
                display: none !important;
            }
            </style>
        """
        st.markdown(hide_img_fs, unsafe_allow_html=True)

        st.write(
            f'**Status**: <span style="float: right;">{self.get_manga_status(manga["Status"])}</span>',
            unsafe_allow_html=True,
        )

        chapter_tag_content = f"""
            <a href="{{}}" target="_blank" style="text-decoration: none; color: {self.default_chapter_link_text_color}">
                <span>{{}}</span>
            </a>
        """
        tagger(
            "<strong>Last Released Chapter:</strong>",
            chapter_tag_content.format(
                manga["LastUploadChapter"]["URL"],
                f'Ch. {manga["LastUploadChapter"]["Chapter"]}',
            ),
            self.default_chapter_link_background_color,
            "float: right;",
        )
        upload_date = get_relative_time(manga["LastUploadChapter"]["UpdatedAt"])
        st.caption(
            f'**Release Date**: <span style="float: right;">{upload_date}</span>',
            unsafe_allow_html=True,
        )

        if manga["LastReadChapter"] is not None:
            tagger(
                "<strong>Last Read Chapter:</strong>",
                chapter_tag_content.format(
                    manga["LastReadChapter"]["URL"],
                    f'Ch. {manga["LastReadChapter"]["Chapter"]}',
                ),
                self.default_chapter_link_background_color,
                "float: right;",
            )
            read_date = get_relative_time(manga["LastReadChapter"]["UpdatedAt"])
            st.caption(
                f'**Read Date**: <span style="float: right;">{read_date}</span>',
                unsafe_allow_html=True,
            )
        else:
            tagger(
                "<strong>Last Read Chapter:</strong>",
                f"""<snap style="text-decoration: none; color: {self.default_chapter_link_text_color}">N/A</span>""",
                self.default_chapter_link_background_color,
                "float: right;",
            )
            st.caption(
                '**Read Date**: <span style="float: right;">N/A</span>',
                unsafe_allow_html=True,
            )

        c1, c2 = st.columns(2)
        with c1:

            def set_last_read():
                if (
                    manga.get("LastReadChapter", {}).get("Chapter")
                    != manga["LastUploadChapter"]["Chapter"]
                ):
                    self.api_client.update_manga_last_read_chapter(
                        manga["LastUploadChapter"]["Chapter"],
                        manga["LastUploadChapter"]["URL"],
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

            def highlight_manga():
                ss["manga_to_highlight"] = manga

            st.button(
                "Highlight",
                use_container_width=True,
                type="primary",
                key=f"highlight_{manga['ID']}",
                on_click=highlight_manga,
            )

    def show_highlighted_manga(self, manga: dict[str, Any]):
        default_size = 36
        characters = len(manga["Name"])
        if characters < 15:
            font_size = default_size
        elif characters < 30:
            font_size = 20
        else:
            font_size = 15
        st.markdown(
            f"""<h1 style='text-align: center; font-size: {font_size}px;'>{manga["Name"]}</h1>""",
            unsafe_allow_html=True,
        )

        # Clean up the last highlighted manga if there is any and set the new manga to highlight
        if ss.get("update_manga_form_manga_name", "") != manga["Name"]:
            with st.spinner("Getting manga metadata..."):
                ss["update_manga_form_manga_name"] = manga["Name"]
                ss["update_manga_form_chapters"] = self.api_client.get_manga_chapters(
                    manga["ID"]
                )
                ss["update_manga_form_default_status_index"] = list(
                    self.manga_status_options.values()
                ).index(str(self.get_manga_status(manga["Status"])))
                ss["update_Manga_form_default_chapter_index"] = list(
                    map(
                        lambda chapter: chapter["Chapter"],
                        ss["update_manga_form_chapters"],
                    )
                ).index(manga["LastReadChapter"]["Chapter"])

            ss["update_manga_form_success_msg"] = None
            ss["update_manga_form_error"] = None

            ss["delete_manga_success_msg"] = None
            ss["delete_manga_error"] = None

        with st.form(key="update_manga_form", border=False):
            st.selectbox(
                "Status",
                index=ss["update_manga_form_default_status_index"],
                options=self.manga_status_options.values(),
                key="update_manga_form_status",
            )

            st.selectbox(
                "Last Read Chapter",
                index=ss["update_Manga_form_default_chapter_index"],
                options=ss["update_manga_form_chapters"],
                key="update_manga_form_chapter",
                format_func=lambda chapter: f"Ch. {chapter['Chapter']}",
            )

            if st.form_submit_button(
                "Update Manga",
                use_container_width=True,
                type="primary",
            ):
                ss["update_manga_form_success_msg"] = None
                ss["update_manga_form_error"] = None
                try:
                    status = ss.update_manga_form_status
                    status = int(self.get_manga_status(status))
                    if status != manga["Status"]:
                        self.api_client.update_manga_status(int(status), manga["ID"])

                    chapter = ss.update_manga_form_chapter
                    if (
                        manga["LastReadChapter"] is None
                        or chapter != manga["LastReadChapter"]["Chapter"]
                    ):
                        self.api_client.update_manga_last_read_chapter(
                            chapter["Chapter"],
                            chapter["URL"],
                            manga["ID"],
                            manga["URL"],
                        )

                    ss["update_manga_form_default_status_index"] = list(
                        self.manga_status_options.values()
                    ).index(ss.update_manga_form_status)
                except Exception as e:
                    ss["update_manga_form_error"] = e
                else:
                    ss["update_manga_form_success_msg"] = "Manga updated successfully"

        def delete_manga_btn_callback():
            ss["delete_manga_error"] = None

            try:
                self.api_client.delete_manga(manga["ID"])
            except Exception as e:
                ss["delete_manga_error"] = e
            else:
                ss["manga_to_highlight"] = None

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

        if ss.get("delete_manga_error", None) is not None:
            st.error(ss["delete_manga_error"])
        if ss.get("update_manga_form_error", None) is not None:
            st.error(ss["update_manga_form_error"])
        if ss.get("update_manga_form_success_msg", None) is not None:
            st.success(ss["update_manga_form_success_msg"])

    def get_manga_status(self, status: int | str) -> str | int:
        if isinstance(status, int):
            return self.manga_status_options[status]
        elif isinstance(status, str):
            for key, value in self.manga_status_options.items():
                if value == status:
                    return key

        st.error(f"Invalid manga status: {status}")
        st.stop()

    def show_add_manga_form(self):
        def get_chapters_btn_callback():
            ss["add_manga_form_default_url"] = ss.add_manga_form_url
            ss["add_manga_url"] = ss.add_manga_form_url

        def add_manga_btn_callback():
            ss["add_manga_chapter"] = ss.add_manga_form_chapter_chapter
            ss["add_manga_form_default_url"] = ""
            ss["add_manga_form_default_status_index"] = 0
            ss["add_manga_form_chapters_options"] = []

        with st.form(
            key="add_manga_form_url_status", border=False, clear_on_submit=True
        ):
            if "add_manga_form_default_url" not in ss:
                ss["add_manga_form_default_url"] = ""
            st.text_input(
                "Manga URL",
                value=ss["add_manga_form_default_url"],
                placeholder="https://mangahub.io/manga/one-piece",
                key="add_manga_form_url",
            )

            get_chapters_btn = st.form_submit_button(
                "Get Chapters", on_click=get_chapters_btn_callback
            )

        if get_chapters_btn:
            # cannnot be in a button callback because it will run before reloading the page
            try:
                with st.spinner("Getting manga chapters..."):
                    ss[
                        "add_manga_form_chapters_options"
                    ] = self.api_client.get_manga_chapters(-1, ss.add_manga_form_url)
            except APIException as e:
                if (
                    "invalid URI for request" in str(e.response_text)
                    or "error getting manga from DB: manga doesn't have an ID or URL"
                    in str(e.response_text)
                ):
                    st.warning("Invalid URL")
                else:
                    st.error(e)

        def status_select_callback():
            ss["add_manga_form_default_status_index"] = list(
                self.manga_status_options.values()
            ).index(ss.add_manga_form_status)

        if "add_manga_form_default_status_index" not in ss:
            ss["add_manga_form_default_status_index"] = 0
        st.selectbox(
            "Status",
            index=ss["add_manga_form_default_status_index"],
            options=self.manga_status_options.values(),
            key="add_manga_form_status",
            on_change=status_select_callback,
        )

        with st.form(key="add_manga_form_chapter", border=False, clear_on_submit=True):
            if ss.get("add_manga_form_chapters_options", None) is None:
                ss["add_manga_form_chapters_options"] = []
            st.selectbox(
                "Last Read Chapter",
                options=ss["add_manga_form_chapters_options"],
                key="add_manga_form_chapter_chapter",
                format_func=lambda chapter: f"Ch. {chapter['Chapter']}",
            )

            if st.form_submit_button("Add Manga", on_click=add_manga_btn_callback):
                add_manga_chapter = ss.get("add_manga_chapter")
                if add_manga_chapter is not None:
                    manga_last_read_chapter = add_manga_chapter["Chapter"]
                    manga_last_read_chapter_url = add_manga_chapter["URL"]
                    manga_status = int(self.get_manga_status(ss.add_manga_form_status))
                    manga_url = ss["add_manga_url"]

                    self.api_client.add_manga(
                        manga_url,
                        manga_status,
                        manga_last_read_chapter,
                        manga_last_read_chapter_url,
                    )
                    st.success("Manga added successfully")
                else:
                    st.error(
                        "Provide a manga URL and select the last read chapter first"
                    )


def main():
    dashboard = MainDashboard()
    dashboard.show()


if __name__ == "__main__":
    logging.basicConfig(
        encoding="utf-8",
        level=logging.INFO,
        format="%(asctime)s :: %(levelname)-8s :: %(name)s :: %(message)s",
    )
    logger = logging.getLogger()

    # Have to be outside the main function
    if st.sidebar.button("Refresh"):
        st.rerun()
    try:
        main()
    except Exception:
        logger.exception("An exception happened!")
        st.error("An error occurred.")

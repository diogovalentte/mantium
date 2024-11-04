import base64
from datetime import datetime
from io import BytesIO
from typing import Any

import src.util.defaults as defaults
import streamlit as st
from PIL import Image
from src.api.api_client import get_api_client
from src.exceptions import APIException
from src.util.add_manga import show_search_manga_term_form
from src.util.util import (
    centered_container,
    get_logger,
    get_relative_time,
    get_source_name_and_colors,
    get_updated_at_datetime,
    tagger,
)
from streamlit import session_state as ss
from streamlit_extras.stylable_container import stylable_container

logger = get_logger()


def show_update_multimanga_form(manga: dict[str, Any]):
    if manga["Source"] == defaults.CUSTOM_MANGA_SOURCE:

        @st.dialog(manga["Name"])
        def show():
            ss["is_dialog_open"] = True
            show_update_custom_manga(manga)

    else:

        @st.dialog(manga["Name"])
        def show():
            ss["is_dialog_open"] = True
            e = st.empty()
            if ss.get("update_multimanga_updated_parts", None) is not None:
                with e.container():
                    updated_parts = ss["update_multimanga_updated_parts"]
                    del ss["update_multimanga_updated_parts"]
                    update_multimanga(updated_parts)
            elif ss.get("update_multimanga_delete_multimanga_id", None) is not None:
                with e.container():
                    id = ss["update_multimanga_delete_multimanga_id"]
                    del ss["update_multimanga_delete_multimanga_id"]
                    delete_manga(id)
            else:
                with st.container():
                    show_update_multimanga(manga["MultiMangaID"])

    show()


def show_update_multimanga_mangas_form(multimanga: dict[str, Any]):
    if ss.get("show_update_multimanga_add_manga_search", False):

        @st.dialog("Add Manga", width="large")
        def show():
            ss["is_dialog_open"] = True
            show_update_multimanga_add_manga_search(multimanga)
            ss["show_update_multimanga_add_manga_search"] = False

    elif ss.get("show_update_multimanga_add_manga_url", False):

        @st.dialog("Add Manga")
        def show():
            ss["is_dialog_open"] = True
            show_update_multimanga_add_manga_url(multimanga)
            ss["show_update_multimanga_add_manga_url"] = False

    else:

        @st.dialog("Manage Mangas", width="large")
        def show():
            ss["is_dialog_open"] = True
            show_update_multimanga_manage_mangas(multimanga)
            ss["show_update_multimanga_manage_mangas"] = False

    show()


def show_update_multimanga(multimanga_id):
    try:
        api_client = get_api_client()
        multimanga = api_client.get_multimanga(multimanga_id)
    except Exception as e:
        logger.exception(e)
        st.error("Error while getting manga")
        st.stop()

    get_chapters_success = False
    get_other_manga_chapters = False
    other_manga_source_name = ""
    try:
        with st.spinner("Getting manga chapters..."):
            ss["update_multimanga_chapter_options"] = (
                api_client.get_cached_manga_chapters(
                    multimanga["CurrentManga"]["ID"],
                    multimanga["CurrentManga"]["URL"],
                    multimanga["CurrentManga"]["InternalID"],
                )
            )
        get_chapters_success = True
    except APIException as e:
        logger.exception(e)
        try:
            with st.spinner("Current manga unavailable. Choosing another manga..."):
                used_ids = []
                used_ids.append(multimanga["CurrentManga"]["ID"])
                for manga in multimanga["Mangas"]:
                    if manga["ID"] == multimanga["CurrentManga"]["ID"]:
                        continue
                    try:
                        manga = api_client.choose_current_manga(
                            multimanga["ID"], exclude_manga_ids=used_ids
                        )
                        ss["update_multimanga_chapter_options"] = (
                            api_client.get_cached_manga_chapters(
                                manga["ID"], manga["URL"], manga["InternalID"]
                            )
                        )
                        get_other_manga_chapters = True
                        other_manga_source_name, _, _ = get_source_name_and_colors(
                            manga["Source"]
                        )
                        break
                    except APIException:
                        logger.exception(e)
                    used_ids.append(manga["ID"])
        except APIException:
            logger.exception(e)

    if not get_chapters_success:
        if get_other_manga_chapters:
            st.warning(
                f"Could not get the current manga's chapters. Maybe the source site is down or the manga URL changed in the source site. Using {other_manga_source_name} chapters instead."
            )
        else:
            ss["update_multimanga_chapter_options"] = []
            st.error(
                "Could not get get chapters from any of the multimanga's mangas. You still can update the other fields."
            )

    with st.form(key="update_multimanga_form", border=False):
        st.selectbox(
            "Status",
            index=multimanga["Status"] - 1,
            options=list(defaults.manga_status_options.keys())[
                1:
            ],  # Exclude the "All" option
            format_func=lambda index: defaults.manga_status_options[index],
            key="update_multimanga_form_status",
        )

        show_chapter_not_found_warning = False
        if (
            multimanga["LastReadChapter"]["Chapter"] != ""
            and ss["update_multimanga_chapter_options"] != []
        ):
            try:
                last_read_chapter_idx = list(
                    map(
                        lambda chapter: chapter["Chapter"],
                        ss["update_multimanga_chapter_options"],
                    )
                ).index(multimanga["LastReadChapter"]["Chapter"])
            except ValueError:
                show_chapter_not_found_warning = True
                last_read_chapter_idx = None
        else:
            last_read_chapter_idx = None
        st.selectbox(
            "Last Read Chapter",
            index=last_read_chapter_idx,
            options=ss["update_multimanga_chapter_options"],
            format_func=lambda chapter: f"Ch. {chapter['Chapter']} --- {get_relative_time(get_updated_at_datetime(chapter['UpdatedAt']))}",
            key="update_multimanga_form_chapter" + str(multimanga["ID"]),
        )

        if show_chapter_not_found_warning:
            st.warning(
                "Last read chapter not found chapters list. Select it again or leave empty to not update it."
            )

        if (
            ss.get("update_multimanga_chapter_options") is not None
            and len(ss.get("update_multimanga_chapter_options", [])) < 1
            and get_chapters_success
        ):
            st.warning(
                "Current manga has no released chapters. You still can update the other fields."
            )

        with st.expander(
            "Update Cover Image",
        ):
            st.info(
                "By default, the cover image of the current manga is used. It's fetched from the manga's source site and it's automatically updated when the source site changes it, but you can manually provide an image URL or upload a file."
            )
            if multimanga["CoverImgFixed"]:
                st.warning(
                    "You changed the cover image of the multimanga to a custom image."
                )
            st.text_input(
                "Cover Image URL",
                placeholder="https://example.com/image.jpg",
                key="update_multimanga_form_cover_img_url",
            )
            st.file_uploader(
                "Upload Cover Image",
                type=["png", "jpg", "jpeg"],
                key="update_multimanga_form_cover_img_upload",
            )
            st.divider()
            st.info(
                "If you manually changed the cover image and want to go back and use the current manga's cover image, check the box below."
            )
            st.checkbox(
                "Use current manga's current image.",
                key="update_multimanga_form_use_current_manga_cover_img",
            )

        if st.form_submit_button(
            "Update Multimanga",
            use_container_width=True,
            type="primary",
        ):
            try:
                cover_url = ss.update_multimanga_form_cover_img_url
                cover_upload = (
                    ss.update_multimanga_form_cover_img_upload.getvalue()
                    if ss.update_multimanga_form_cover_img_upload
                    else None
                )
                use_current_manga_cover_img = (
                    ss.update_multimanga_form_use_current_manga_cover_img
                )

                values_count = sum(
                    [
                        bool(cover_url),
                        bool(cover_upload),
                        use_current_manga_cover_img,
                    ]
                )

                if values_count > 1:
                    ss["update_manga_warning_message"] = (
                        "To update the cover image, provide either an URL, upload a file, or check the box to use the current manga's cover image."
                    )
                else:
                    ss["update_multimanga_updated_parts"] = {}
                    ss["update_multimanga_updated_parts"]["multimanga"] = multimanga

                    ss["update_multimanga_updated_parts"][
                        "status"
                    ] = ss.update_multimanga_form_status

                    ss["update_multimanga_updated_parts"]["chapter"] = ss[
                        "update_multimanga_form_chapter" + str(multimanga["ID"])
                    ]

                    if values_count == 1:
                        ss["update_multimanga_updated_parts"]["update_cover"] = True
                        ss["update_multimanga_updated_parts"]["cover_url"] = cover_url
                        ss["update_multimanga_updated_parts"][
                            "cover_upload"
                        ] = cover_upload
                        ss["update_multimanga_updated_parts"][
                            "use_current_manga_cover_img"
                        ] = use_current_manga_cover_img

                    else:
                        ss["update_multimanga_updated_parts"]["update_cover"] = False

                if ss.get("update_manga_warning_message", "") == "":
                    st.rerun(scope="fragment")
            except APIException as e:
                logger.exception(e)
                ss["update_manga_error_message"] = "Error while updating multimanga"

    with stylable_container(
        key="update_multimanga_delete_button",
        css_styles="""
            button {
                background-color: red;
                color: white;
            }
        """,
    ):

        def on_click():
            ss["update_multimanga_delete_multimanga_id"] = multimanga["ID"]

        st.button(
            "Delete Multimanga",
            use_container_width=True,
            on_click=on_click,
        )

    if ss.get("update_manga_warning_message", "") != "":
        st.warning(ss["update_manga_warning_message"])
    if ss.get("update_manga_error_message", "") != "":
        st.error(ss["update_manga_error_message"])
    ss["update_manga_warning_message"] = ""
    ss["update_manga_error_message"] = ""

    st.divider()

    with st.expander("Multimanga Mangas"):

        if st.button(
            "Add Manga by Searching",
            use_container_width=True,
            type="primary",
            key="update_multimanga_mangas_show_add_manga_search_button",
        ):
            ss["show_update_multimanga_add_manga_search"] = True
            ss["highlighted_multimanga"] = multimanga
            st.rerun()

        if st.button(
            "Add Manga Using URL",
            use_container_width=True,
            type="primary",
            key="update_multimanga_mangas_show_add_manga_url_button",
        ):
            ss["show_update_multimanga_add_manga_url"] = True
            ss["highlighted_multimanga"] = multimanga
            st.rerun()

        if st.button(
            "Manage Multimanga Mangas",
            use_container_width=True,
            type="primary",
            key="update_multimanga_mangas_show_manage_mangas_button",
        ):
            ss["show_update_multimanga_manage_mangas"] = True
            ss["highlighted_multimanga"] = multimanga
            st.rerun()


def update_multimanga(updated_parts):
    api_client = get_api_client()
    st.info("Updating manga...")
    try:
        multimanga = updated_parts["multimanga"]
        status = updated_parts["status"]
        if status != multimanga["Status"]:
            api_client.update_multimanga_status(status, multimanga["ID"])

        chapter = updated_parts["chapter"]
        if chapter is not None and (
            multimanga["LastReadChapter"] is None
            or chapter["URL"] != multimanga["LastReadChapter"]["URL"]
            or chapter["Chapter"] != multimanga["LastReadChapter"]["Chapter"]
        ):
            api_client.update_multimanga_last_read_chapter(
                multimanga["ID"],
                multimanga["CurrentManga"]["ID"],
                chapter["Chapter"],
                chapter["URL"],
                chapter["InternalID"],
            )

        if updated_parts["update_cover"]:
            cover_url = updated_parts.get("cover_url", "")
            cover_upload = updated_parts.get("cover_upload", None)
            use_current_manga_cover_img = updated_parts.get(
                "use_current_manga_cover_img", False
            )

            if cover_url != "" or cover_upload is not None:
                api_client.update_multimanga_cover_img(
                    multimanga["ID"],
                    cover_img_url=cover_url,
                    cover_img=cover_upload if cover_upload else b"",
                )
            elif use_current_manga_cover_img:
                api_client.update_multimanga_cover_img(
                    multimanga["ID"],
                    use_current_manga_cover_img=use_current_manga_cover_img,
                )
    except Exception as ex:
        logger.exception(ex)
        st.error("Error while updating manga")
        st.stop()
    else:
        ss["update_manga_success_message"] = "Multimanga updated successfully"
        st.rerun()


def delete_manga(multimanga_id: int):
    st.info("Deleting manga...")
    try:
        with st.spinner("Deleting multimanga..."):
            api_client = get_api_client()
            api_client.delete_multimanga(multimanga_id)
    except Exception as e:
        logger.exception(e)
        st.error("Error while deleting multimanga")
        st.stop()
    else:
        ss["update_manga_success_message"] = "Multimanga deleted successfully"
        st.rerun()


def show_update_multimanga_add_manga_search(multimanga):
    api_client = get_api_client()
    button_name, key_to_save_manga = (
        "Add Manga",
        "update_multimanga_mangas_add_manga_selected_manga",
    )

    container = st.empty()
    with container:
        if ss.get(key_to_save_manga) is not None:
            manga = ss[key_to_save_manga]
            ss[key_to_save_manga] = None
            try:
                with st.spinner("Adding manga to multimanga..."):
                    api_client.add_manga_to_multimanga(
                        multimanga["ID"],
                        manga["URL"],
                        manga["InternalID"],
                    )
            except APIException as e:
                if "manga already exists in DB" in str(e):
                    st.warning(
                        "Manga already exists in multimanga or as a normal manga"
                    )
                else:
                    logger.exception(e)
                    st.error("Error while adding manga to multimanga")
            else:
                ss["update_manga_success_message"] = (
                    "Manga added to multimanga successfully"
                )
                st.rerun()
        else:
            (
                mangadex_tab,
                comick_tab,
                mangaplus_tab,
                mangahub_tab,
                mangaupdates_tab,
            ) = st.tabs(["Mangadex", "Comick", "Mangaplus", "Mangahub", "MangaUpdates"])

            base_key = key_to_save_manga + "_search_results"
            ss[base_key + "_mangadex"] = {}
            ss[base_key + "_comick"] = {}
            ss[base_key + "_mangaplus"] = {}
            ss[base_key + "_mangahub"] = {}
            ss[base_key + "_mangaupdates"] = {}

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


def show_update_multimanga_add_manga_url(multimanga):
    api_client = get_api_client()
    manga_url = st.text_input(
        "Manga URL",
        placeholder="https://mangahub.io/manga/one-piece",
        key="update_multimanga_mangas_add_manga_url_provided_url",
    )
    if manga_url != "":
        try:
            with st.spinner("Adding manga to multimanga..."):
                api_client.add_manga_to_multimanga(
                    multimanga["ID"],
                    manga_url,
                    "",
                )
        except APIException as e:
            resp_text = str(e.response_text).lower()
            if "manga already exists in db" in resp_text:
                st.warning("Manga already exists in multimanga or as a normal manga")
            elif (
                "error while getting source: source '" in resp_text
                and "not found" in resp_text
            ):
                st.warning("No source site for this manga")
            elif "invalid manga url" in resp_text:
                st.warning("Invalid URL")
            elif "manga not found in source" in resp_text:
                st.warning("Manga not found")
            else:
                logger.exception(e)
                st.error("Error while adding manga to multimanga")
        else:
            ss["update_manga_success_message"] = (
                "Manga added to multimanga successfully"
            )
            st.rerun()


def show_update_multimanga_manage_mangas(multimanga):
    message_container = st.empty()

    mangas = multimanga["Mangas"]
    cols_list = st.columns(2)
    show_multimanga_mangas(
        cols_list, mangas, multimanga["CurrentManga"]["ID"], multimanga["ID"]
    )

    if st.button(
        "Back",
        key="update_multimanga_mangas_manage_mangas_back_button",
        use_container_width=True,
    ):
        ss["highlighted_manga"] = multimanga["CurrentManga"]
        st.rerun()

    with message_container.container():
        if ss.get("update_manga_error_message", "") != "":
            st.error(ss["update_manga_error_message"])
            ss["update_manga_error_message"] = ""
        if ss.get("update_manga_warning_message", "") != "":
            st.warning(ss["update_manga_warning_message"])
            ss["update_manga_warning_message"] = ""


def show_multimanga_mangas(
    cols_list: list, mangas, current_manga_id: int, multimanga_id: int
):
    """Show the mangas of a multimanga in the cols_list columns.

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
                    show_multimanga_manga(
                        manga, multimanga_id, manga["ID"] == current_manga_id
                    )
        col_index += 1


def show_multimanga_manga(
    manga: dict[str, Any], multimanga_id: int, current_manga: bool = False
):
    api_client = get_api_client()
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

    chapter_tag_content = f"""
        <a href="{manga["LastReleasedChapter"]["URL"]}" target="_blank" style="text-decoration: none; color: {defaults.chapter_link_tag_text_color}">
            <span>{f'Ch. {manga["LastReleasedChapter"]["Chapter"]}' if manga["LastReleasedChapter"]["Chapter"] != "" else "N/A"}</span>
        </a>
    """

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
        chapter_tag_content,
        defaults.chapter_link_tag_background_color,
        "float: right;",
    )
    st.caption(
        f'**Release Date**: <span style="float: right;" title="{release_date}">{relative_release_date}</span>',
        unsafe_allow_html=True,
    )

    source_name, source_text_color, source_background_color = (
        get_source_name_and_colors(manga["Source"])
    )

    tags = f"""
        <span style="display:inline-block;
        background-color: {source_background_color};
        padding: 0.1rem 0.5rem;
        font-size: 14px;
        font-weight: 400;
        color: {source_text_color};
        border-radius: 1rem;">{source_name}</span>
    """
    if current_manga:
        tags += f"""<span style="display:inline-block;
            background-color: {defaults.chapter_link_tag_background_color};
            padding: 0.1rem 0.5rem;
            font-size: 14px;
            font-weight: 400;
            color: {defaults.chapter_link_tag_text_color};
            border-radius: 1rem;
            float: right;">Current Manga</span>
        """

    st.write(tags, unsafe_allow_html=True)

    with stylable_container(
        key="update_manga_delete_button",
        css_styles="""
            button {
                background-color: red;
                color: white;
            }
        """,
    ):
        if st.button(
            "Remove",
            key="update_multimanga_mangas_mangage_mangas_" + str(manga["ID"]),
            use_container_width=True,
        ):
            try:
                api_client.remove_manga_from_multimanga(multimanga_id, manga["ID"])
            except Exception as e:
                if (
                    "attempted to remove the last manga from a multimanga"
                    in str(e).lower()
                ):
                    ss["update_manga_warning_message"] = (
                        "Can't remove the last manga from a multimanga. Delete the multimanga instead"
                    )
                else:
                    logger.exception(e)
                    ss["update_manga_error_message"] = "Error while removing manga"
            else:
                ss["update_manga_success_message"] = "Manga removed successfully"
            if not (
                ss.get("update_manga_error_message", "") != ""
                or ss.get("update_manga_warning_message", "") != ""
            ):
                st.rerun()


def show_update_custom_manga(manga: dict[str, Any]):
    api_client = get_api_client()

    with st.form(key="update_custom_manga_form", border=False):
        st.text_input(
            "Manga Name (not optional)",
            value=manga["Name"],
            placeholder="One Piece",
            key="update_custom_manga_form_name",
        )

        st.text_input(
            "Manga URL",
            value=manga["URL"],
            placeholder="https://randomsite.com/title/one-piece",
            key="update_custom_manga_form_url",
        )

        st.selectbox(
            "Status",
            index=manga["Status"] - 1,
            options=list(defaults.manga_status_options.keys())[
                1:
            ],  # Exclude the "All" option
            format_func=lambda index: defaults.manga_status_options[index],
            key="update_custom_manga_form_status",
        )

        with st.expander(
            "Next Chapter",
        ):
            st.text_input(
                "Next Chapter to Read",
                value=manga["LastReadChapter"]["Chapter"],
                placeholder="1000",
                help="Can be a number or text",
                key="update_custom_manga_form_chapter",
            )

            st.text_input(
                "Chapter URL",
                value=manga["LastReadChapter"]["URL"],
                placeholder="https://randomsite.com/title/one-piece/chapter/1000",
                key="update_custom_manga_form_chapter_url",
            )

            st.checkbox(
                "No more chapters available",
                value=manga["LastReadChapter"]["Chapter"]
                == manga["LastReleasedChapter"]["Chapter"],
                help="Check this if there are no more chapters available. By default, if next chapter is empty, it's checked, even if you changed it previously. You can change it anytime.",
                key="update_custom_manga_form_no_more_chapters",
            )

        with st.expander(
            "Cover Image",
        ):
            st.text_input(
                "Cover Image URL",
                placeholder="https://example.com/image.jpg",
                key="update_custom_manga_form_cover_img_url",
            )
            st.file_uploader(
                "Upload Cover Image",
                type=["png", "jpg", "jpeg"],
                key="update_custom_manga_form_cover_img_file",
            )
            st.checkbox(
                "Use Mantium default cover image",
                key="update_custom_manga_form_use_mantim_default_cover_img",
            )

        if st.form_submit_button(
            "Update Manga",
            use_container_width=True,
            type="primary",
        ):
            try:
                name = ss.update_custom_manga_form_name
                url = ss.update_custom_manga_form_url
                status = ss.update_custom_manga_form_status
                next_chapter = ss.update_custom_manga_form_chapter
                next_chapter_url = ss.update_custom_manga_form_chapter_url
                manga_has_more_chapters = (
                    not ss.update_custom_manga_form_no_more_chapters
                )
                cover_img_url = ss.update_custom_manga_form_cover_img_url
                cover_img = (
                    ss.update_custom_manga_form_cover_img_file.getvalue()
                    if ss.update_custom_manga_form_cover_img_file
                    else None
                )
                use_mantium_default_cover_img = (
                    ss.update_custom_manga_form_use_mantim_default_cover_img
                )
                if name == "":
                    st.warning("Provide a manga name")
                elif next_chapter == "" and next_chapter_url != "":
                    ss["update_manga_warning_message"] = (
                        "Provide a chapter number to go with the chapter URL"
                    )
                else:
                    if name != manga["Name"]:
                        api_client.update_manga_name(name, manga["ID"])

                    if url != manga["URL"]:
                        api_client.update_manga_url(url, manga["ID"])

                    if status != manga["Status"]:
                        api_client.update_manga_status(status, manga["ID"])

                    if (
                        next_chapter != manga["LastReadChapter"]["Chapter"]
                        or next_chapter_url != manga["LastReadChapter"]["URL"]
                    ):
                        api_client.update_manga_last_read_chapter(
                            manga["ID"],
                            "",
                            "",
                            next_chapter,
                            next_chapter_url,
                            "",
                        )

                    if (
                        (
                            manga_has_more_chapters
                            and manga["LastReadChapter"]["Chapter"] != ""
                            and manga["LastReadChapter"]["Chapter"]
                            == manga["LastReleasedChapter"]["Chapter"]
                        )
                        or (
                            not manga_has_more_chapters
                            and manga["LastReadChapter"]["Chapter"] != ""
                            and manga["LastReadChapter"]["Chapter"]
                            != manga["LastReleasedChapter"]["Chapter"]
                        )
                        or next_chapter != manga["LastReadChapter"]["Chapter"]
                    ):
                        api_client.update_custom_manga_has_more_chapters(
                            manga_has_more_chapters, manga["ID"], ""
                        )

                    values_count = sum(
                        [
                            bool(cover_img_url),
                            bool(cover_img),
                            use_mantium_default_cover_img,
                        ]
                    )
                    match values_count:
                        case 0:
                            pass
                        case 1:
                            if cover_img_url != "" or cover_img is not None:
                                api_client.update_manga_cover_img(
                                    manga["ID"],
                                    "",
                                    "",
                                    cover_img_url,
                                    cover_img if cover_img else b"",
                                    False,
                                    False,
                                )
                            elif use_mantium_default_cover_img:
                                api_client.update_manga_cover_img(
                                    manga["ID"],
                                    "",
                                    "",
                                    "",
                                    b"",
                                    False,
                                    True,
                                )
                        case _:
                            ss["update_manga_warning_message"] = (
                                "To update the cover image, provide either an URL, upload a file, or check the box to use the Mantium default cover image. The other fields were updated successfully"
                            )

                    if not (
                        ss.get("update_manga_error_message", "") != ""
                        or ss.get("update_manga_warning_message", "") != ""
                    ):
                        ss["update_manga_success_message"] = (
                            "Manga updated successfully"
                        )
                        st.rerun()
            except Exception as e:
                logger.exception(e)
                ss["update_manga_error_message"] = "Error while updating manga"

    with stylable_container(
        key="update_custom_manga_delete_button",
        css_styles="""
            button {
                background-color: red;
                color: white;
            }
        """,
    ):
        if st.button(
            "Delete Manga",
            use_container_width=True,
        ):
            try:
                api_client.delete_manga(manga["ID"])
            except Exception as e:
                logger.exception(e)
                ss["update_manga_error_message"] = "Error while deleting manga"
            else:
                ss["update_manga_success_message"] = "Manga deleted successfully"
            if not (
                ss.get("update_manga_error_message", "") != ""
                or ss.get("update_manga_warning_message", "") != ""
            ):
                st.rerun()

    if ss.get("update_manga_error_message", "") != "":
        st.error(ss["update_manga_error_message"])
    if ss.get("update_manga_warning_message", "") != "":
        st.warning(ss["update_manga_warning_message"])
    ss["update_manga_error_message"] = ""
    ss["update_manga_warning_message"] = ""

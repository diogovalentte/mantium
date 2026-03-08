import base64
from datetime import datetime, timezone
from io import BytesIO
from typing import Any

import src.util.defaults as defaults
import streamlit as st
from PIL import Image
from src.api.api_client import get_api_client
from src.exceptions import APIException
from src.util.add_manga import show_add_custom_manga_form, show_search_manga_term_form
from src.util.util import (
    centered_container,
    get_logger,
    get_relative_time,
    get_source_name_and_colors,
    get_updated_at_datetime,
    tagger,
    set_is_dialog_open,
)
from streamlit import session_state as ss
from streamlit_extras.stylable_container import stylable_container

logger = get_logger()


def show_update_multimanga_form(manga: dict[str, Any]):
    @st.dialog(manga["Name"], on_dismiss=set_is_dialog_open)
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
                delete_multimanga(id)
        else:
            with st.container():
                show_update_multimanga(manga["MultiMangaID"])

    show()


def show_update_multimanga_mangas_form(multimanga: dict[str, Any]):
    if ss.get("show_update_multimanga_add_manga_search", False):

        @st.dialog("Add Manga", width="medium", on_dismiss=set_is_dialog_open)
        def show():
            ss["is_dialog_open"] = True
            show_update_multimanga_add_manga_search(multimanga)
            ss["show_update_multimanga_add_manga_search"] = False

    elif ss.get("show_update_multimanga_add_manga_url", False):

        @st.dialog("Add Manga", on_dismiss=set_is_dialog_open)
        def show():
            ss["is_dialog_open"] = True
            show_update_multimanga_add_manga_url(multimanga)
            ss["show_update_multimanga_add_manga_url"] = False

    elif ss.get("show_update_multimanga_add_custom_manga", False):

        @st.dialog("Add Manga", on_dismiss=set_is_dialog_open)
        def show():
            ss["is_dialog_open"] = True
            show_update_multimanga_add_custom_manga(multimanga)
            ss["show_update_multimanga_add_custom_manga"] = False

    else:

        @st.dialog("Manage Mangas", width="medium", on_dismiss=set_is_dialog_open)
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
        st.error("Error while getting multimanga")
        st.stop()

    custom_manga_ids = []
    other_manga_source_name = ""
    if "update_multimanga_chapter_options" in ss:
        del ss["update_multimanga_chapter_options"]
        del ss["update_multimanga_chapter_options_from_manga_id"]

    for manga in multimanga["Mangas"]:
        if manga["Source"] == defaults.CUSTOM_MANGA_SOURCE:
            custom_manga_ids.append(manga["ID"])

    if len(multimanga["Mangas"]) != len(custom_manga_ids):
        with st.spinner("Getting manga chapters..."):
            used_ids = custom_manga_ids.copy()
            for manga in multimanga["Mangas"]:
                if manga["Source"] == defaults.CUSTOM_MANGA_SOURCE:
                    continue
                try:
                    manga_to_get_chapters = api_client.choose_current_manga(multimanga["ID"], exclude_manga_ids=used_ids)
                    ss["update_multimanga_chapter_options"] = (
                        api_client.get_cached_manga_chapters(
                            manga_to_get_chapters["ID"],
                            manga_to_get_chapters["URL"],
                            manga_to_get_chapters["InternalID"],
                        )
                    )
                    ss["update_multimanga_chapter_options_from_manga_id"] = manga_to_get_chapters["ID"]
                    if multimanga["CurrentManga"]["ID"] != manga_to_get_chapters["ID"]:
                        other_manga_source_name = get_source_name_and_colors(manga_to_get_chapters["Source"])[0]
                    break
                except APIException as e:
                    logger.exception(e)
                finally:
                    used_ids.append(manga["ID"])

    with st.form(key="update_multimanga_form", border=False):
        st.selectbox(
            "Status",
            index=multimanga["Status"] - 1,
            options=list(defaults.manga_status_options.keys())[
                1:
            ],  # Exclude the "All" option
            format_func=lambda index: defaults.manga_status_options[index],
            key="update_multimanga_form_status_" + str(multimanga["ID"]),
        )

        with st.expander("Last Read Chapter"):
            if ss.get("update_multimanga_chapter_options", []) == [] and len(multimanga["Mangas"]) != len(custom_manga_ids):
                # Could have fetched the chapters but failed or got an empty chapters list
                if ss.get("update_multimanga_chapter_options") is None:
                    # Failed to get the chapters from any manga
                    st.warning(
                        "Could not get the chapters list from the multimanga mangas. Maybe the source sites are down or the manga URLs changed in the source sites."
                    )
                elif ss.get("update_multimanga_chapter_options") == []:
                    # Managed to get an empty chapters list
                    if other_manga_source_name == "":
                        st.warning(
                            "No released chapters found."
                        )
                    else:
                        st.warning(
                            f"Could not get the chapters list from the current manga. Maybe the source site is down or the manga URL changed in the source site. Others manga returned empty chapters list."
                        )
            elif ss.get("update_multimanga_chapter_options", []) != [] and other_manga_source_name != "" and multimanga["CurrentManga"]["Source"] != defaults.CUSTOM_MANGA_SOURCE:
                st.warning(
                    f"Could not get the chapters list from the current manga. Maybe the source site is down or the manga URL changed in the source site. Chapters list fetched from the manga of source {other_manga_source_name}."
                )

            if (
                multimanga["LastReadChapter"]["Chapter"] != ""
                and ss.get("update_multimanga_chapter_options", []) != []
                and multimanga["LastReadChapter"]["FromSourceSite"]
            ):
                try:
                    last_read_chapter_idx = list(
                        map(
                            lambda chapter: chapter["Chapter"],
                            ss["update_multimanga_chapter_options"],
                        )
                    ).index(multimanga["LastReadChapter"]["Chapter"])
                except ValueError:
                    last_read_chapter_idx = None
            else:
                last_read_chapter_idx = None

            st.selectbox(
                "From chapters list",
                help="Select the chapter from the chapters list if you want to update it to a chapter that was released from source site. If the chapter you want to update to is not in the chapters list, write it manually in the text input bellow.",
                index=last_read_chapter_idx,
                options=ss.get("update_multimanga_chapter_options", []),
                format_func=lambda chapter: f"Ch. {chapter['Chapter']}{(' (' + get_relative_time(get_updated_at_datetime(chapter['UpdatedAt']))) + ')' if chapter['UpdatedAt'] != '0001-01-01T00:00:00Z' else ''}",
                key="update_multimanga_form_chapter_from_chapters_list_" + str(multimanga["ID"]),
            )

            if last_read_chapter_idx is None and multimanga["LastReadChapter"]["Chapter"] != "" and multimanga["LastReadChapter"]["FromSourceSite"] and ss.get("update_multimanga_chapter_options", []) != []:
                st.warning(
                    "Last read chapter not found in chapters list. Select it again, leave empty or write it manually bellow."
                )

            st.divider()

            st.text_input(
                "Last Read Chapter",
                value=multimanga["LastReadChapter"]["Chapter"] if not multimanga["LastReadChapter"]["FromSourceSite"] else "",
                placeholder="1000",
                help="Can be a number or text",
                key="update_multimanga_form_chapter_manual_chapter_" + str(multimanga["ID"]),
            )

            st.text_input(
                "Chapter URL",
                help="Cannot set only the URL without the chapter, chapter field must be filled. If chapter is not filled, set to the last released chapter.",
                value=multimanga["LastReadChapter"]["URL"] if not multimanga["LastReadChapter"]["FromSourceSite"] else "",
                placeholder="https://randomsite.com/title/one-piece/chapter/1000",
                key="update_multimanga_form_chapter_manual_url_" + str(multimanga["ID"]),
            )

            st.checkbox(
                "Delete Last Read Chapter",
                help="Delete the last read chapter and set it to empty. This will make Mantium consider that you haven't read any chapter.",
                key="update_multimanga_form_delete_last_read_chapter_" + str(multimanga["ID"]),
            )

        with st.expander(
            "Cover Image",
        ):
            st.info(
                "By default, the cover image of the current manga is used. It's fetched from the manga's source site and it's automatically updated when the source site changes it, but you can manually provide an image URL or upload a file."
            )
            if multimanga["CoverImgFixed"]:
                st.warning("The cover image is set to a custom image provided by you.")
            st.text_input(
                "Cover Image URL",
                placeholder="https://example.com/image.jpg",
                key="update_multimanga_form_cover_img_url_" + str(multimanga["ID"]),
            )
            st.file_uploader(
                "Upload Cover Image",
                type=["png", "jpg", "jpeg"],
                key="update_multimanga_form_cover_img_upload_" + str(multimanga["ID"]),
            )
            st.divider()
            st.info(
                "If you manually changed the cover image and want to go back and use the current manga image, check the box below."
            )
            st.checkbox(
                "Use current manga image",
                key="update_multimanga_form_use_current_manga_cover_img_"
                + str(multimanga["ID"]),
            )

        if st.form_submit_button(
            "Update",
            use_container_width=True,
            type="primary",
        ):
            try:
                cover_url = ss[
                    "update_multimanga_form_cover_img_url_" + str(multimanga["ID"])
                ]
                cover_upload = (
                    ss[
                        "update_multimanga_form_cover_img_upload_"
                        + str(multimanga["ID"])
                    ].getvalue()
                    if ss[
                        "update_multimanga_form_cover_img_upload_"
                        + str(multimanga["ID"])
                    ]
                    else None
                )
                use_current_manga_cover_img = ss[
                    "update_multimanga_form_use_current_manga_cover_img_"
                    + str(multimanga["ID"])
                ]

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

                    ss["update_multimanga_updated_parts"]["status"] = ss[
                        "update_multimanga_form_status_" + str(multimanga["ID"])
                    ]

                    ss["update_multimanga_updated_parts"]["chapter_from_chapters_list"] = ss[
                        "update_multimanga_form_chapter_from_chapters_list_" + str(multimanga["ID"])
                    ]
                    ss["update_multimanga_updated_parts"]["chapter_from_chapters_list_from_manga_id"] = ss.get("update_multimanga_chapter_options_from_manga_id", 0)
                    ss["update_multimanga_updated_parts"]["chapter_manual"] = {
                        "chapter": ss["update_multimanga_form_chapter_manual_chapter_" + str(multimanga["ID"])],
                        "url": ss["update_multimanga_form_chapter_manual_url_" + str(multimanga["ID"])]
                    }
                    ss["update_multimanga_updated_parts"]["delete_last_read_chapter"] = ss[
                        "update_multimanga_form_delete_last_read_chapter_" + str(multimanga["ID"])
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
                ss["update_manga_error_message"] = "Error while updating"

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
            "Delete",
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
            "Add Custom Manga",
            use_container_width=True,
            type="primary",
            key="update_multimanga_mangas_show_add_custom_manga_button",
        ):
            ss["show_update_multimanga_add_custom_manga"] = True
            ss["highlighted_multimanga"] = multimanga
            st.rerun()

        if st.button(
            "Manage Mangas",
            use_container_width=True,
            type="primary",
            key="update_multimanga_mangas_show_manage_mangas_button",
        ):
            ss["show_update_multimanga_manage_mangas"] = True
            ss["highlighted_multimanga"] = multimanga
            st.rerun()


def update_multimanga(updated_parts):
    api_client = get_api_client()
    st.info("Updating...")
    try:
        multimanga = updated_parts["multimanga"]
        status = updated_parts["status"]
        if status != multimanga["Status"]:
            api_client.update_multimanga_status(status, multimanga["ID"])

        last_read_chapter = multimanga["LastReadChapter"]
        chapter_from_chapters_list = updated_parts["chapter_from_chapters_list"]
        chapter_manual = updated_parts["chapter_manual"]
        delete_last_read_chapter = updated_parts["delete_last_read_chapter"]

        if delete_last_read_chapter:
            api_client.update_multimanga_last_read_chapter(
                multimanga["ID"],
                multimanga["CurrentManga"]["ID"],
                "",
                "",
                "",
                True,
                False
            )
        elif chapter_from_chapters_list is not None and (
            last_read_chapter is None
            or chapter_from_chapters_list["Chapter"] != last_read_chapter["Chapter"]
        ):
            api_client.update_multimanga_last_read_chapter(
                multimanga["ID"],
                updated_parts["chapter_from_chapters_list_from_manga_id"],
                chapter_from_chapters_list["Chapter"],
                chapter_from_chapters_list["URL"],
                chapter_from_chapters_list["InternalID"],
                False,
                True
            )
        elif (chapter_manual["chapter"] != "" or chapter_manual["url"] != "") and (
            last_read_chapter is None
            or chapter_manual["chapter"] != last_read_chapter["Chapter"]
            or chapter_manual["url"] != last_read_chapter["URL"]
        ):
            api_client.update_multimanga_last_read_chapter(
                multimanga["ID"],
                multimanga["CurrentManga"]["ID"],
                chapter_manual["chapter"],
                chapter_manual["url"],
                "",
                False,
                False
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


def delete_multimanga(multimanga_id: int):
    st.info("Deleting...")
    try:
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
                        "",
                        manga["URL"],
                        manga["InternalID"],
                        "",
                        b"",
                        "",
                        "",
                        "",
                        "",
                        "",
                        "",
                        "",
                        False
                    )
            except APIException as e:
                if "manga already exists in DB" in str(e):
                    st.warning("Manga already exists")
                elif "source" in str(e).lower() and "is not allowed" in str(e).lower():
                    st.warning("Not allowed to add mangas from this source")
                else:
                    logger.exception(e)
                    st.error("Error while adding manga to multimanga")
            else:
                ss["update_manga_success_message"] = "Manga added successfully"
                st.rerun()
        else:
            sources = {}
            for name, source in defaults.default_sources.items():
                if source in ss["configs"]["manga"]["allowedSources"]:
                    sources[name] = defaults.default_sources[name]

            base_key = key_to_save_manga + "_search_results"
            for source in list(sources.values()):
                ss[base_key + "_" + source] = {}
            tabs = st.tabs(list(sources.keys()))
            for i, source in enumerate(sources.keys()):
                with tabs[i]:
                    show_search_manga_term_form(
                        sources[source], button_name, key_to_save_manga
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
                    "",
                    manga_url,
                    "",
                    "",
                    b"",
                    "",
                    "",
                    "",
                    "",
                    "",
                    "",
                    "",
                    False
                )
        except APIException as e:
            resp_text = str(e.response_text).lower()
            if "manga already exists in db" in resp_text:
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
                st.error("Error while adding manga to multimanga")
        else:
            ss["update_manga_success_message"] = "Manga added successfully"
            st.rerun()

def show_update_multimanga_add_custom_manga(multimanga):
    show_add_custom_manga_form(multimanga["ID"])

def show_update_multimanga_manage_mangas(multimanga):
    message_container = st.empty()

    mangas = multimanga["Mangas"]
    columns_num = 2 if not ss["is_mobile"] else 1
    cols_list = st.columns(columns_num)
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
                with centered_container("center_container" + str(manga["ID"])):
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
            class="manga_header" style='padding-top: 16px; text-align: center; margin-top: {margin}px; margin-bottom: {margin}px; font-size: {font_size}px;'>
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
            f"""<img src="{manga["CoverImgURL"]}" width="250" height="355"/>""",
            unsafe_allow_html=True,
        )
    else:
        st.markdown(
            f"""<img src="{defaults.DEFAULT_MANGA_COVER}" width="250" height="355"/>""",
            unsafe_allow_html=True,
        )

    chapter_tag_content = f"""
        <a href="{manga["LastReleasedChapter"]["URL"]}" target="_blank" style="text-decoration: none; color: {defaults.chapter_link_tag_text_color}">
            <span>{f'Ch. {manga["LastReleasedChapter"]["Chapter"]}' if manga["LastReleasedChapter"]["Chapter"] != "" else "N/A"}</span>
        </a>
    """

    release_date = (
        manga["LastReleasedChapter"]["UpdatedAt"]
        if manga["LastReleasedChapter"]["UpdatedAt"] != datetime.min.replace(tzinfo=timezone.utc)
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

    if manga["Source"] == defaults.CUSTOM_MANGA_SOURCE:
        def on_click():
            ss["highlighted_custom_manga"] = manga

        if st.button(
            "Edit",
            key="update_multimanga_mangas_edit_custom_manga_button_" + str(manga["ID"]),
            use_container_width=True,
            type="primary",
            on_click=on_click,
        ):
            st.rerun()

    with stylable_container(
        key="update_manga_delete_button" + str(manga["ID"]),
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
            help="The URL of the manga in the source site. It's used by the Last Released Chapter selectors to check for new chapters if you provided the selectors. If you have to change both the URL and selectors to make them work, DON'T update them at the same time. Clear the selectors, update the URL, then update the selectors.",
            key="update_custom_manga_form_url",
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
                    value=manga["LastReleasedChapterNameSelector"]["Selector"],
                    placeholder="css:div.chapter-list > a:nth-child(1) > div.chapter-title",
                    help="CSS or XPath selector to get the chapter name. E.g. 'css:div.chapter-list > a:nth-child(1) > div.chapter-title' or 'xpath://div[@class=\"chapter-list\"]//a[1]//div[@class=\"chapter-title\"]'. Leave empty to skip.",
                    key="update_custom_manga_form_last_chapter_name_selector",
                )
                st.text_input(
                    "Attribute",
                    value=manga["LastReleasedChapterNameSelector"]["Attribute"],
                    placeholder="href",
                    help="Element attribute to get the chapter name. E.g. 'href' for link. Leave empty to get the inner text.",
                    key="update_custom_manga_form_last_chapter_name_attribute",
                )
                st.text_input(
                    "Regex",
                    value=manga["LastReleasedChapterNameSelector"]["Regex"],
                    placeholder="Chapter (\\d+)",
                    help="Regex to extract the chapter name. E.g. 'Chapter (\\d+)' to extract '100' from 'Chapter 100'. Leave empty to skip.",
                    key="update_custom_manga_form_last_chapter_name_regex",
                )
                st.checkbox(
                    "Get First",
                    value=manga["LastReleasedChapterNameSelector"]["GetFirst"],
                    help="Get the first match instead of the last match. Useful for sites that list chapters in ascending order.",
                    key="update_custom_manga_form_last_released_chapter_name_get_first",
                )
            with st.expander(
                "Chapter URL"
            ):
                st.text_input(
                    "Selector",
                    value=manga["LastReleasedChapterURLSelector"]["Selector"],
                    placeholder="css:div.chapter-list > a:nth-child(1)",
                    help="CSS or XPath selector to get the chapter URL. E.g. 'css:div.chapter-list > a:nth-child(1)' or 'xpath://div[@class=\"chapter-list\"]//a[1]'. Leave empty to skip.",
                    key="update_custom_manga_form_last_chapter_url_selector",
                )
                st.text_input(
                    "Attribute",
                    value=manga["LastReleasedChapterURLSelector"]["Attribute"],
                    placeholder="href",
                    help="Element attribute to get the chapter URL. E.g. 'href' for link. Leave empty to get the inner text.",
                    key="update_custom_manga_form_last_chapter_url_attribute",
                )
                st.checkbox(
                    "Get First",
                    value=manga["LastReleasedChapterURLSelector"]["GetFirst"],
                    help="Get the first match instead of the last match. Useful for sites that list chapters in ascending order.",
                    key="update_custom_manga_form_last_released_chapter_url_get_first",
                )

            st.checkbox(
                "Use Browser",
                value=manga["LastReleasedChapterSelectorUseBrowser"],
                help="Use a browser to fetch the manga page. Use this if the chapter info is loaded dynamically with JavaScript. Requires more resources and can take longer.",
                key="update_custom_manga_form_last_released_chapter_use_browser",
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
            "Update",
            use_container_width=True,
            type="primary",
        ):
            try:
                name = ss.update_custom_manga_form_name
                url = ss.update_custom_manga_form_url

                last_released_chapter_name_selector = ss.update_custom_manga_form_last_chapter_name_selector
                last_released_chapter_name_attribute = ss.update_custom_manga_form_last_chapter_name_attribute
                last_released_chapter_name_regex = ss.update_custom_manga_form_last_chapter_name_regex
                last_released_chapter_name_get_first = ss.update_custom_manga_form_last_released_chapter_name_get_first
                last_released_chapter_url_selector = ss.update_custom_manga_form_last_chapter_url_selector
                last_released_chapter_url_attribute = ss.update_custom_manga_form_last_chapter_url_attribute
                last_released_chapter_url_get_first = ss.update_custom_manga_form_last_released_chapter_url_get_first
                last_released_chapter_selector_use_browser = ss.update_custom_manga_form_last_released_chapter_use_browser

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
                else:
                    if name != manga["Name"]:
                        api_client.update_custom_manga_name(name, manga["ID"])

                    if url != manga["URL"]:
                        api_client.update_custom_manga_url(url, manga["ID"])

                    if (
                        last_released_chapter_name_selector
                        != manga["LastReleasedChapterNameSelector"]["Selector"]
                        or last_released_chapter_name_attribute
                        != manga["LastReleasedChapterNameSelector"]["Attribute"]
                        or last_released_chapter_name_regex
                        != manga["LastReleasedChapterNameSelector"]["Regex"]
                        or last_released_chapter_name_get_first
                        != manga["LastReleasedChapterNameSelector"]["GetFirst"]
                        or last_released_chapter_url_selector
                        != manga["LastReleasedChapterURLSelector"]["Selector"]
                        or last_released_chapter_url_attribute
                        != manga["LastReleasedChapterURLSelector"]["Attribute"]
                        or last_released_chapter_url_get_first
                        != manga["LastReleasedChapterURLSelector"]["GetFirst"]
                        or last_released_chapter_selector_use_browser
                        != manga["LastReleasedChapterSelectorUseBrowser"]
                    ):
                        api_client.update_custom_manga_last_released_chapter_selectors(
                            last_released_chapter_name_selector,
                            last_released_chapter_name_attribute,
                            last_released_chapter_name_regex,
                            last_released_chapter_name_get_first,
                            last_released_chapter_url_selector,
                            last_released_chapter_url_attribute,
                            last_released_chapter_url_get_first,
                            last_released_chapter_selector_use_browser,
                            manga["ID"],
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
                                api_client.update_custom_manga_cover_img(
                                    manga["ID"],
                                    cover_img_url,
                                    cover_img if cover_img else b"",
                                    False,
                                )
                            elif use_mantium_default_cover_img:
                                api_client.update_custom_manga_cover_img(
                                    manga["ID"],
                                    "",
                                    b"",
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
                        ss["highlighted_custom_manga"] = None
                        st.rerun()
            except Exception as ex:
                resp_text = str(ex).lower()
                logger.exception(ex)
                if "error while visiting manga url" in resp_text:
                    ss["update_manga_warning_message"] = (
                        "Error while visiting the provided manga URL. Check that it's valid."
                    )
                elif "selector not found in the page or is empty" in resp_text or "selector should start with 'css:' or 'xpath:', instead it's" in resp_text:
                    ss["update_manga_warning_message"] = (
                        "Invalid selector or one of the provided selectors/attributes was not found in the page or is empty"
                    )
                elif "regex did not match" in resp_text or "error compiling regex" in resp_text:
                    ss["update_manga_warning_message"] = (
                        "Invalid regex or the chapter name selector is empty after applying the provided regex"
                    )
                else:
                    ss["update_manga_error_message"] = "Error while updating manga"

    if ss.get("update_manga_error_message", "") != "":
        st.error(ss["update_manga_error_message"])
    if ss.get("update_manga_warning_message", "") != "":
        st.warning(ss["update_manga_warning_message"])
    ss["update_manga_error_message"] = ""
    ss["update_manga_warning_message"] = ""

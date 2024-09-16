from typing import Any

import src.util.defaults as defaults
import streamlit as st
from src.api.api_client import get_api_client
from src.exceptions import APIException
from src.util.util import get_logger, get_manga_chapters, get_relative_time
from streamlit import session_state as ss
from streamlit_extras.stylable_container import stylable_container

logger = get_logger()


def show_update_manga(manga: dict[str, Any]):
    api_client = get_api_client()
    try:
        with st.spinner("Getting manga chapters..."):
            ss["update_manga_chapter_options"] = get_manga_chapters(
                manga["ID"], manga["URL"], manga["InternalID"]
            )
    except APIException as e:
        logger.exception(e)
        st.error("Error while getting manga chapters")
        st.stop()

    with st.form(key="update_manga_form", border=False):
        st.selectbox(
            "Status",
            index=manga["Status"] - 1,
            options=list(defaults.manga_status_options.keys())[
                1:
            ],  # Exclude the "All" option
            format_func=lambda index: defaults.manga_status_options[index],
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
            last_read_chapter_idx = len(ss["update_manga_chapter_options"]) - 1
        st.selectbox(
            "Last Read Chapter",
            index=last_read_chapter_idx,
            options=ss["update_manga_chapter_options"],
            format_func=lambda chapter: f"Ch. {chapter['Chapter']} --- {get_relative_time(api_client.get_updated_at_datetime(chapter['UpdatedAt']))}",
            key="update_manga_form_chapter" + str(manga["ID"]),
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
                    api_client.update_manga_status(status, manga["ID"])

                chapter = ss["update_manga_form_chapter" + str(manga["ID"])]
                if chapter is not None and (
                    manga["LastReadChapter"] is None
                    or chapter["URL"] != manga["LastReadChapter"]["URL"]
                    or chapter["Chapter"] != manga["LastReadChapter"]["Chapter"]
                ):
                    api_client.update_manga_last_read_chapter(
                        manga["ID"],
                        manga["URL"],
                        manga["InternalID"],
                        chapter["Chapter"],
                        chapter["URL"],
                        chapter["InternalID"],
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
                            api_client.update_manga_cover_img(
                                manga["ID"],
                                manga["URL"],
                                manga["InternalID"],
                                cover_img_url=cover_url,
                                cover_img=cover_upload if cover_upload else b"",
                            )
                        elif get_cover_img_from_source:
                            api_client.update_manga_cover_img(
                                manga["ID"],
                                manga["URL"],
                                manga["InternalID"],
                                get_cover_img_from_source=get_cover_img_from_source,
                            )
                    case _:
                        ss["update_manga_warning_message"] = (
                            "To update the cover image, provide either an URL, upload a file, or check the box to get the image from the source site. The other fields were updated successfully."
                        )
                if not (
                    ss.get("update_manga_error_message", "") != ""
                    or ss.get("update_manga_warning_message", "") != ""
                ):
                    ss["update_manga_success_message"] = "Manga updated successfully"
                    st.rerun()
            except APIException as e:
                logger.exception(e)
                ss["update_manga_error_message"] = "Error while updating manga."

    def delete_manga_btn_callback():
        try:
            api_client.delete_manga(manga["ID"])
        except Exception as e:
            logger.exception(e)
            ss["update_manga_error_message"] = "Error while deleting manga."
        else:
            ss["update_manga_success_message"] = "Manga deleted successfully"

    with stylable_container(
        key="highlight_manga_delete_button",
        css_styles="""
            button {
                background-color: red;
                color: white;
            }
        """,
    ):
        if st.button(
            "Delete Manga",
            on_click=delete_manga_btn_callback,
            use_container_width=True,
        ):
            if not (
                ss.get("update_manga_error_message", "") != ""
                or ss.get("update_manga_warning_message", "") != ""
            ):
                st.rerun()
    if (
        ss.get("update_manga_error_message", "") != ""
        or ss.get("update_manga_warning_message", "") != ""
    ):
        if ss.get("update_manga_error_message", "") != "":
            st.error(ss["update_manga_error_message"])
        if ss.get("update_manga_warning_message", "") != "":
            st.warning(ss["update_manga_warning_message"])
        ss["update_manga_error_message"] = ""
        ss["update_manga_warning_message"] = ""

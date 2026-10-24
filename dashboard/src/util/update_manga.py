from datetime import datetime, timezone
from typing import Any

import src.util.defaults as defaults
import streamlit as st
from src.api.api_client import get_api_client
from src.exceptions import APIException
from src.util.add_manga import (
    CUSTOM_MANGA_URL_LIVE,
    get_custom_manga_selector_values,
    on_custom_manga_url_change,
    reset_custom_manga_form_state,
    run_pending_custom_manga_preset_detection,
    show_custom_manga_tracking_fields,
    show_add_custom_manga_form,
    show_search_manga_term_form,
)
from src.util.util import (
    URL_INPUT_VALIDATION,
    centered_container,
    escape_html,
    get_api_error_detail,
    get_api_user_message,
    get_invalid_url_message,
    get_url_hostname,
    get_logger,
    get_relative_time,
    get_source_name_and_colors,
    get_updated_at_datetime,
    safe_href,
    show_manga_cover,
    tagger,
    set_is_dialog_open,
)
from streamlit import session_state as ss
from streamlit_extras.stylable_container import stylable_container

logger = get_logger()


def reset_update_multimanga_chapters():
    """The chapters are fetched again each time the dialog opens, but not on each
    rerun inside it."""
    for key in ("update_multimanga_chapters", "update_multimanga_chapter_options", "update_multimanga_chapter_options_from_manga_id"):
        ss.pop(key, None)


def show_update_multimanga_form(manga: dict[str, Any]):
    reset_update_multimanga_chapters()

    @st.dialog(manga["Name"], on_dismiss=set_is_dialog_open)
    def show():
        ss["is_dialog_open"] = True
        show_update_multimanga(manga["MultiMangaID"])

    show()


def show_update_multimanga_mangas_form(multimanga: dict[str, Any]):
    if ss.get("show_update_multimanga_add_manga_search", False):
        # Reset when the dialog opens, not on each rerun inside it, so the results
        # aren't searched again on every click
        for source in defaults.default_sources.values():
            ss["update_multimanga_mangas_add_manga_selected_manga_search_results_" + source] = {}

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


def show_back_to_multimanga_button(multimanga: dict[str, Any], key: str):
    """Goes back from a multimanga mangas form to the multimanga form."""
    if st.button("Back", key=key, width="stretch"):
        # st.rerun() skips the lines that reset these after the form is shown
        for form in ("add_manga_search", "add_manga_url", "add_custom_manga", "manage_mangas"):
            ss["show_update_multimanga_" + form] = False
        ss["highlighted_manga"] = multimanga["CurrentManga"]
        st.rerun()


def custom_manga_lists_chapters(manga: dict[str, Any]) -> bool:
    """Tells if a manga is a custom manga whose selectors can get its chapters list, which needs a chapter name selector."""
    if manga["Source"] != defaults.CUSTOM_MANGA_SOURCE:
        return False
    name_selector = manga.get("LastReleasedChapterNameSelector") or {}
    return name_selector.get("Selector", "") != ""


def get_multimanga_chapters_options(
    api_client, multimanga: dict[str, Any], custom_manga_ids: list[int], current_manga_lists_chapters: bool
) -> list[dict[str, Any]]:
    """Gets the chapters list shown in the Last Read Chapter field into session_state.

    Tries the current manga first, then the other non-custom mangas, until one returns
    chapters. Returns what happened with each manga tried, for chapters_list_warning.
    The options are None if every manga failed, and [] if the ones that answered had no chapters.
    """
    attempts = []

    def got_chapters(manga: dict[str, Any], chapters: list[dict[str, Any]]) -> bool:
        attempts.append(chapters_attempt(manga, "ok" if chapters else "empty"))
        if ss.get("update_multimanga_chapter_options") is None or chapters:
            ss["update_multimanga_chapter_options"] = chapters
        if chapters:
            ss["update_multimanga_chapter_options_from_manga_id"] = manga["ID"]
        return bool(chapters)

    if current_manga_lists_chapters:
        current_manga = multimanga["CurrentManga"]
        try:
            # Not cached: the list changes with the custom manga selectors
            chapters = api_client.get_manga_chapters(current_manga["ID"], current_manga["URL"], current_manga["InternalID"])
        except APIException as e:
            logger.exception(e)
            attempts.append(chapters_attempt(current_manga, "error", get_api_user_message(e) or get_api_error_detail(e)))
        else:
            if got_chapters(current_manga, chapters):
                return attempts

    used_ids = custom_manga_ids.copy()
    # One attempt per non-custom manga: each candidate the API hands us
    # is excluded from the next attempt, so a source that fails or has no
    # chapters is never picked twice and every source gets tried once.
    for _ in range(len(multimanga["Mangas"]) - len(custom_manga_ids)):
        manga_to_get_chapters = None
        try:
            manga_to_get_chapters = api_client.choose_current_manga(multimanga["ID"], exclude_manga_ids=used_ids)
            chapters = api_client.get_cached_manga_chapters(
                manga_to_get_chapters["ID"],
                manga_to_get_chapters["URL"],
                manga_to_get_chapters["InternalID"],
            )
        except APIException as e:
            logger.exception(e)
            if manga_to_get_chapters is not None:
                attempts.append(chapters_attempt(manga_to_get_chapters, "error"))
            else:
                # Couldn't even choose which manga to try
                attempts.append({"manga_id": None, "custom": False, "source_name": "", "result": "error", "error": None})
        else:
            if got_chapters(manga_to_get_chapters, chapters):
                break
        finally:
            if manga_to_get_chapters is not None:
                used_ids.append(manga_to_get_chapters["ID"])

    return attempts


def chapters_attempt(manga: dict[str, Any], result: str, error: str | None = None) -> dict[str, Any]:
    """What happened when getting a manga chapters list: result is "ok", "empty" or "error"."""
    return {
        "manga_id": manga["ID"],
        "custom": manga["Source"] == defaults.CUSTOM_MANGA_SOURCE,
        "source_name": get_source_name_and_colors(manga["Source"])[0],
        "result": result,
        "error": error,
    }


def chapters_list_warning(current_manga_id: int, attempts: list[dict[str, Any]], from_manga_id: int | None) -> str | None:
    """The warning about where the chapters list of the Last Read Chapter field came from, or why there's none."""
    current = next((a for a in attempts if a["manga_id"] == current_manga_id), None)
    others = [a for a in attempts if a["manga_id"] != current_manga_id]

    current_problem = None
    if current is not None:
        if current["result"] == "empty":
            current_problem = "The current manga has no released chapters."
        elif current["result"] == "error" and current["custom"]:
            current_problem = f"Could not get the chapters list from the current custom manga: {current['error'].rstrip('.')}."
        elif current["result"] == "error":
            current_problem = "Could not get the chapters list from the current manga. Maybe the source site is down or the manga URL changed in the source site."

    if from_manga_id is not None:
        if from_manga_id == current_manga_id or current_problem is None:
            return None
        source_name = next(a["source_name"] for a in attempts if a["manga_id"] == from_manga_id)
        return f"{current_problem} Chapters list fetched from the manga of source {source_name}."

    if not attempts:
        return None
    results = {a["result"] for a in others}
    if current is None:
        # The current manga can't list chapters, so only the other mangas were tried
        if results == {"empty"}:
            return "No released chapters found."
        if results == {"error"}:
            return "Could not get the chapters list from the multimanga mangas. Maybe the source sites are down or the manga URLs changed in the source sites."
        return "Could not get the chapters list from the multimanga mangas: they failed or have no released chapters."
    if not others:
        return "No released chapters found." if current["result"] == "empty" else current_problem
    if results == {"empty"}:
        others_problem = "The other mangas have no released chapters either."
    elif results == {"error"}:
        others_problem = "Could not get the chapters list from the other mangas either. Maybe their source sites are down or the manga URLs changed in the source sites."
    else:
        others_problem = "The other mangas failed or have no released chapters either."
    return f"{current_problem} {others_problem}"


def show_update_multimanga(multimanga_id):
    try:
        api_client = get_api_client()
        multimanga = api_client.get_multimanga(multimanga_id)
    except Exception as e:
        logger.exception(e)
        st.error("Error while getting multimanga")
        st.stop()

    custom_manga_ids = []
    for manga in multimanga["Mangas"]:
        if manga["Source"] == defaults.CUSTOM_MANGA_SOURCE:
            custom_manga_ids.append(manga["ID"])
    current_manga_lists_chapters = custom_manga_lists_chapters(multimanga["CurrentManga"])
    gets_chapters = current_manga_lists_chapters or len(multimanga["Mangas"]) != len(custom_manga_ids)

    cached = ss.get("update_multimanga_chapters")
    if cached is not None and cached["multimanga_id"] == multimanga["ID"]:
        chapters_attempts = cached["attempts"]
    else:
        chapters_attempts = []
        ss.pop("update_multimanga_chapter_options", None)
        ss.pop("update_multimanga_chapter_options_from_manga_id", None)
        if gets_chapters:
            with st.spinner("Getting manga chapters..."):
                chapters_attempts = get_multimanga_chapters_options(api_client, multimanga, custom_manga_ids, current_manga_lists_chapters)
        ss["update_multimanga_chapters"] = {
            "multimanga_id": multimanga["ID"],
            "attempts": chapters_attempts,
        }

    st.selectbox(
        "Status",
        index=multimanga["Status"] - 1,
        options=list(defaults.manga_status_options.keys())[
            1:
        ],  # Exclude the "All" option
        format_func=lambda index: defaults.manga_status_options[index],
        key="update_multimanga_form_status_" + str(multimanga["ID"]),
    )

    reset_last_read_chapter = ss.get("update_multimanga_form_delete_last_read_chapter_" + str(multimanga["ID"]), False)

    with st.expander("Last Read Chapter"):
        if warning := chapters_list_warning(
            multimanga["CurrentManga"]["ID"],
            chapters_attempts,
            ss.get("update_multimanga_chapter_options_from_manga_id"),
        ):
            st.warning(warning)

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
            help="Select the chapter from the chapters list if you want to update it to a chapter that was released from source site. If the chapter you want to update to is not in the chapters list, write it manually in the text input below.",
            index=last_read_chapter_idx,
            options=ss.get("update_multimanga_chapter_options", []),
            format_func=lambda chapter: f"Ch. {chapter['Chapter']}{(' (' + get_relative_time(get_updated_at_datetime(chapter['UpdatedAt']))) + ')' if chapter['UpdatedAt'] != '0001-01-01T00:00:00Z' else ''}",
            disabled=reset_last_read_chapter,
            key="update_multimanga_form_chapter_from_chapters_list_" + str(multimanga["ID"]),
        )

        if last_read_chapter_idx is None and multimanga["LastReadChapter"]["Chapter"] != "" and multimanga["LastReadChapter"]["FromSourceSite"] and ss.get("update_multimanga_chapter_options", []) != []:
            st.warning(
                "Last read chapter not found in chapters list. Select it again, leave it empty, or write it manually below."
            )

        st.divider()

        st.text_input(
            "Last Read Chapter",
            value=multimanga["LastReadChapter"]["Chapter"] if not multimanga["LastReadChapter"]["FromSourceSite"] else "",
            placeholder="1000",
            help="Can be a number or text",
            disabled=reset_last_read_chapter,
            key="update_multimanga_form_chapter_manual_chapter_" + str(multimanga["ID"]),
        )

        st.text_input(
            "Chapter URL",
            help="Cannot set only the URL without the chapter, chapter field must be filled. If chapter is not filled, set to the last released chapter.",
            value=multimanga["LastReadChapter"]["URL"] if not multimanga["LastReadChapter"]["FromSourceSite"] else "",
            placeholder="https://randomsite.com/title/one-piece/chapter/1000",
            type="url",
            validate=URL_INPUT_VALIDATION,
            disabled=reset_last_read_chapter,
            key="update_multimanga_form_chapter_manual_url_" + str(multimanga["ID"]),
        )

        st.checkbox(
            "Reset Last Read Chapter",
            help="Reset the last read chapter. This will make Mantium consider that you haven't read any chapter. The fields above are ignored while it's checked.",
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
            type="url",
            validate=URL_INPUT_VALIDATION,
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
        if count_multimanga_cover_values(multimanga["ID"]) > 1:
            st.warning(MULTIMANGA_COVER_CONFLICT_MESSAGE)

    update_clicked = st.button(
        "Update",
        width="stretch",
        type="primary",
        key="update_multimanga_form_update_" + str(multimanga["ID"]),
    )
    with stylable_container(
        key="update_multimanga_delete_button",
        css_styles="""
            button {
                background-color: red;
                color: white;
            }
        """,
    ):
        with st.popover("Delete", width="stretch"):
            st.warning(f'Delete "{multimanga["CurrentManga"]["Name"]}" and its {len(multimanga["Mangas"])} manga(s) from Mantium?')
            if st.button("Yes, delete", width="stretch", key="update_multimanga_form_confirm_delete_" + str(multimanga["ID"])):
                try:
                    with st.spinner("Deleting..."):
                        get_api_client().delete_multimanga(multimanga["ID"])
                except Exception as ex:
                    logger.exception(ex)
                    st.error(f"Error while deleting multimanga: {get_api_error_detail(ex)}")
                else:
                    ss["update_manga_success_message"] = "Multimanga deleted successfully"
                    st.rerun()

    # Below the buttons, so the message doesn't move them
    message_slot = st.empty()
    if update_clicked:
        if count_multimanga_cover_values(multimanga["ID"]) > 1:
            message_slot.warning(MULTIMANGA_COVER_CONFLICT_MESSAGE)
        else:
            with st.spinner("Updating..."):
                error = update_multimanga(multimanga)
            if error:
                message_slot.error(error)
            else:
                ss["update_manga_success_message"] = "Multimanga updated successfully"
                st.rerun()

    st.divider()

    with st.expander("Multimanga Mangas"):

        if st.button(
            "Add Manga by Searching",
            width="stretch",
            type="primary",
            key="update_multimanga_mangas_show_add_manga_search_button",
        ):
            ss["show_update_multimanga_add_manga_search"] = True
            ss["highlighted_multimanga"] = multimanga
            st.rerun()

        if st.button(
            "Add Manga Using URL",
            width="stretch",
            type="primary",
            key="update_multimanga_mangas_show_add_manga_url_button",
        ):
            ss["show_update_multimanga_add_manga_url"] = True
            ss["highlighted_multimanga"] = multimanga
            st.rerun()

        if st.button(
            "Add Custom Manga",
            width="stretch",
            type="primary",
            key="update_multimanga_mangas_show_add_custom_manga_button",
        ):
            ss["show_update_multimanga_add_custom_manga"] = True
            ss["highlighted_multimanga"] = multimanga
            st.rerun()

        if st.button(
            "Manage Mangas",
            width="stretch",
            type="primary",
            key="update_multimanga_mangas_show_manage_mangas_button",
        ):
            ss["show_update_multimanga_manage_mangas"] = True
            ss["highlighted_multimanga"] = multimanga
            st.rerun()


MULTIMANGA_COVER_CONFLICT_MESSAGE = "To update the cover image, provide either an URL, upload a file, or check the box to use the current manga's cover image."


def count_multimanga_cover_values(multimanga_id: int) -> int:
    key = "update_multimanga_form_%s_" + str(multimanga_id)
    return sum([
        bool(ss.get(key % "cover_img_url")),
        bool(ss.get(key % "cover_img_upload")),
        bool(ss.get(key % "use_current_manga_cover_img")),
    ])


def update_multimanga(multimanga: dict[str, Any]) -> str:
    """Updates the multimanga from the form fields. Returns an error message, empty
    if everything was updated. The parts are updated one by one, so the message
    says which ones were updated before the error."""
    api_client = get_api_client()
    key = "update_multimanga_form_%s_" + str(multimanga["ID"])
    updated = []
    part = "status"
    try:
        status = ss[key % "status"]
        if status != multimanga["Status"]:
            api_client.update_multimanga_status(status, multimanga["ID"])
            updated.append("status")

        part = "last read chapter"
        last_read_chapter = multimanga["LastReadChapter"]
        chapter_from_chapters_list = ss[key % "chapter_from_chapters_list"]
        chapter_manual = {"chapter": ss[key % "chapter_manual_chapter"], "url": ss[key % "chapter_manual_url"]}

        if ss[key % "delete_last_read_chapter"]:
            api_client.update_multimanga_last_read_chapter(
                multimanga["ID"],
                multimanga["CurrentManga"]["ID"],
                "",
                "",
                "",
                True,
                False
            )
            updated.append("last read chapter")
        elif chapter_from_chapters_list is not None and (
            last_read_chapter is None
            or chapter_from_chapters_list["Chapter"] != last_read_chapter["Chapter"]
        ):
            api_client.update_multimanga_last_read_chapter(
                multimanga["ID"],
                ss.get("update_multimanga_chapter_options_from_manga_id", 0),
                chapter_from_chapters_list["Chapter"],
                chapter_from_chapters_list["URL"],
                chapter_from_chapters_list["InternalID"],
                False,
                True
            )
            updated.append("last read chapter")
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
            updated.append("last read chapter")

        part = "cover image"
        cover_url = ss[key % "cover_img_url"]
        cover_upload = ss[key % "cover_img_upload"].getvalue() if ss[key % "cover_img_upload"] else None
        if cover_url != "" or cover_upload is not None:
            api_client.update_multimanga_cover_img(
                multimanga["ID"],
                cover_img_url=cover_url,
                cover_img=cover_upload if cover_upload else b"",
            )
        elif ss[key % "use_current_manga_cover_img"]:
            api_client.update_multimanga_cover_img(
                multimanga["ID"],
                use_current_manga_cover_img=True,
            )
    except Exception as ex:
        logger.exception(ex)
        message = f"Error while updating the {part}: {get_api_error_detail(ex)}"
        if updated:
            message += f". Already updated before the error: {', '.join(updated)}."
        return message
    return ""


def show_update_multimanga_add_manga_search(multimanga):
    api_client = get_api_client()
    button_name, key_to_save_manga = (
        "Add Manga",
        "update_multimanga_mangas_add_manga_selected_manga",
    )

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
            # Shown below the search term of the manga's source, and the search
            # results are kept to pick another manga
            if "manga already exists in DB" in str(e):
                message = ("warning", "Manga already exists")
            elif "source" in str(e).lower() and "is not allowed" in str(e).lower():
                message = ("warning", "Not allowed to add mangas from this source")
            else:
                logger.exception(e)
                message = ("error", "Error while adding manga to multimanga")
            ss[key_to_save_manga + "_message"] = (manga["Source"], *message)
        else:
            ss["update_manga_success_message"] = "Manga added successfully"
            st.rerun()

    sources = {}
    for name, source in defaults.default_sources.items():
        if source in ss["configs"]["manga"]["allowedSources"]:
            sources[name] = defaults.default_sources[name]

    tabs = st.tabs(list(sources.keys()))
    for i, source in enumerate(sources.keys()):
        with tabs[i]:
            show_search_manga_term_form(
                sources[source], button_name, key_to_save_manga
            )

    show_back_to_multimanga_button(multimanga, "update_multimanga_mangas_add_manga_search_back_button")

def show_update_multimanga_add_manga_url(multimanga):
    api_client = get_api_client()
    manga_url = st.text_input(
        "Manga URL",
        placeholder="https://mangahub.io/manga/one-piece",
        type="url",
        validate=URL_INPUT_VALIDATION,
        key="update_multimanga_mangas_add_manga_url_provided_url",
    )
    # A button, so a typo isn't added as soon as the field loses focus
    add_clicked = st.button(
        "Add",
        type="primary",
        width="stretch",
        key="update_multimanga_mangas_add_manga_url_add_button",
    )
    show_back_to_multimanga_button(multimanga, "update_multimanga_mangas_add_manga_url_back_button")
    # Below the buttons, so the message doesn't move them
    message_slot = st.empty()
    if add_clicked:
        with message_slot.container():
            if manga_url == "":
                st.warning("Provide the manga URL")
            else:
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
    show_add_custom_manga_form(
        multimanga["ID"],
        lambda: show_back_to_multimanga_button(multimanga, "update_multimanga_mangas_add_custom_manga_back_button"),
    )

def show_update_multimanga_manage_mangas(multimanga):
    mangas = multimanga["Mangas"]
    columns_num = 2 if not ss["is_mobile"] else 1
    cols_list = st.columns(columns_num)
    show_multimanga_mangas(
        cols_list, mangas, multimanga["CurrentManga"]["ID"], multimanga["ID"]
    )

    show_back_to_multimanga_button(multimanga, "update_multimanga_mangas_manage_mangas_back_button")


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
                with centered_container("show_multimanga_mangas_center_container" + str(manga["ID"])):
                    show_multimanga_manga(
                        manga, multimanga_id, manga["ID"] == current_manga_id, len(mangas) > 1
                    )
        col_index += 1


def show_multimanga_manga(
    manga: dict[str, Any], multimanga_id: int, current_manga: bool = False, can_ignore_chapter: bool = False
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
                color: #089a8d;
            }
        </style>
    """
    st.markdown(improve_headers, unsafe_allow_html=True)
    st.markdown(
        f"""<h1
            class="manga_header" style='padding-top: 16px; text-align: center; margin-top: {margin}px; margin-bottom: {margin}px; font-size: {font_size}px;'>
                <a class="manga_header" href="{safe_href(manga["URL"])}" target="_blank">{escape_html(manga["Name"])}</a>
            </h1>
        """,
        unsafe_allow_html=True,
    )

    show_manga_cover(manga, (250, 355))

    chapter_tag_content = f"""
        <a href="{safe_href(manga["LastReleasedChapter"]["URL"])}" target="_blank" style="text-decoration: none; color: {defaults.chapter_link_tag_text_color}">
            <span>{"Ch. " + escape_html(manga["LastReleasedChapter"]["Chapter"]) if manga["LastReleasedChapter"]["Chapter"] != "" else "N/A"}</span>
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
    ignored_chapter = manga.get("IgnoredChapter")
    if ignored_chapter:
        tags += f"""<span style="display:inline-block;
            background-color: {defaults.ignored_chapter_tag_background_color};
            padding: 0.1rem 0.5rem;
            font-size: 14px;
            font-weight: 400;
            color: {defaults.ignored_chapter_tag_text_color};
            border-radius: 1rem;
            float: right;
            margin-right: 0.3rem;">Ch. {escape_html(ignored_chapter["Chapter"] or "N/A")} ignored</span>
        """

    st.write(tags, unsafe_allow_html=True)

    if manga["Source"] == defaults.CUSTOM_MANGA_SOURCE:
        def on_click():
            ss["highlighted_custom_manga"] = manga
            ss["update_custom_manga_back_to_multimanga_id"] = multimanga_id

        if st.button(
            "Edit",
            key="update_multimanga_mangas_edit_custom_manga_button_" + str(manga["ID"]),
            width="stretch",
            type="primary",
            on_click=on_click,
        ):
            st.rerun()

    if ignored_chapter:
        show_stop_ignoring_chapter_popover(manga, multimanga_id)
    elif can_ignore_chapter and (manga.get("LastReleasedChapter") or {}).get("URL"):
        show_ignore_chapter_popover(manga, multimanga_id)

    with stylable_container(
        key="update_manga_delete_button" + str(manga["ID"]),
        css_styles=f"""
            button {{
                background-color: red;
                color: white;
            }}

            .st-key-st-key-update_manga_delete_button{manga['ID']} > div:first-child {{
                margin-bottom: 0rem !important;
            }}
        """,
    ):
        with st.popover("Remove", width="stretch"):
            st.warning(f'Remove "{manga["Name"]}" from this multimanga?')
            if st.button(
                "Yes, remove",
                key="update_multimanga_mangas_mangage_mangas_" + str(manga["ID"]),
                width="stretch",
            ):
                try:
                    api_client.remove_manga_from_multimanga(multimanga_id, manga["ID"])
                    # Fetched again, so the Manage Mangas form reopens without the manga
                    multimanga = api_client.get_multimanga(multimanga_id)
                except Exception as e:
                    if (
                        "attempted to remove the last manga from a multimanga"
                        in str(e).lower()
                    ):
                        st.warning(
                            "Can't remove the last manga from a multimanga. Delete the multimanga instead"
                        )
                    else:
                        logger.exception(e)
                        st.error(f"Error while removing manga: {get_api_error_detail(e)}")
                else:
                    ss["update_manga_success_message"] = "Manga removed successfully"
                    ss["highlighted_multimanga"] = multimanga
                    st.rerun()


def show_ignore_chapter_popover(manga: dict[str, Any], multimanga_id: int):
    chapter = manga["LastReleasedChapter"]["Chapter"] or "N/A"
    source_name, _, _ = get_source_name_and_colors(manga["Source"])
    with st.popover("Ignore Last Chapter", width="stretch"):
        st.warning(
            f"Ignore chapter {chapter} from {source_name}? Use it when the source released a wrong chapter. "
            "The multimanga will follow its other mangas until this source releases another chapter."
        )
        clicked = st.button(
            "Yes, ignore",
            key="update_multimanga_mangas_ignore_chapter_" + str(manga["ID"]),
            width="stretch",
        )
        # Fixed slot, so an error doesn't stay below the button on the dialog's next rerun
        message = st.empty()
        if clicked:
            update_multimanga_manga_ignored_chapter(manga, multimanga_id, True, message)


def show_stop_ignoring_chapter_popover(manga: dict[str, Any], multimanga_id: int):
    with st.popover("Stop Ignoring Chapter", width="stretch"):
        st.warning(
            f"Stop ignoring chapter {manga['IgnoredChapter']['Chapter'] or 'N/A'}? "
            "It'll count again for the multimanga's last released chapter."
        )
        clicked = st.button(
            "Yes, stop ignoring",
            key="update_multimanga_mangas_stop_ignoring_chapter_" + str(manga["ID"]),
            width="stretch",
        )
        message = st.empty()
        if clicked:
            update_multimanga_manga_ignored_chapter(manga, multimanga_id, False, message)


def update_multimanga_manga_ignored_chapter(manga: dict[str, Any], multimanga_id: int, ignore: bool, message):
    api_client = get_api_client()
    try:
        if ignore:
            api_client.ignore_multimanga_manga_chapter(multimanga_id, manga["ID"])
        else:
            api_client.stop_ignoring_multimanga_manga_chapter(multimanga_id, manga["ID"])
        # Fetched again, so the Manage Mangas form reopens with the new current manga
        multimanga = api_client.get_multimanga(multimanga_id)
    except Exception as e:
        if user_message := get_api_user_message(e):
            message.warning(user_message)
        else:
            logger.exception(e)
            message.error(f"Error while updating the ignored chapter: {get_api_error_detail(e)}")
    else:
        ss["update_manga_success_message"] = "Chapter ignored successfully" if ignore else "Chapter is no longer ignored"
        ss["highlighted_multimanga"] = multimanga
        st.rerun()


def show_update_custom_manga(manga: dict[str, Any]):
    api_client = get_api_client()
    key_prefix = "update_custom_manga_form_"

    name_selector = manga["LastReleasedChapterNameSelector"]
    url_selector = manga["LastReleasedChapterURLSelector"]
    reset_custom_manga_form_state(
        key_prefix,
        {
            "name": manga["Name"],
            "url": manga["URL"],
            "last_released_chapter_name_selector": name_selector["Selector"],
            "last_released_chapter_name_attribute": name_selector["Attribute"],
            "last_released_chapter_name_regex": name_selector["Regex"],
            "last_released_chapter_name_get_first": name_selector["GetFirst"],
            "last_released_chapter_url_selector": url_selector["Selector"],
            "last_released_chapter_url_attribute": url_selector["Attribute"],
            "last_released_chapter_url_get_first": url_selector["GetFirst"],
            "last_released_chapter_use_browser": manga["LastReleasedChapterSelectorUseBrowser"],
            "check_url": manga.get("CheckURL", ""),
            "last_released_chapter_expander": bool(name_selector["Selector"] or url_selector["Selector"]),
            "new_preset_domains": get_url_hostname(manga.get("CheckURL") or manga["URL"]),
        },
    )

    with st.container(border=False):
        st.text_input(
            "Manga Name (not optional)",
            placeholder="One Piece",
            key=key_prefix + "name",
        )

        st.text_input(
            "Manga URL",
            placeholder="https://randomsite.com/title/one-piece",
            help="The URL of the manga in the source site. The chapter selectors check it for new chapters, unless a check URL is set. You can change the URL and the selectors at the same time.",
            type="url",
            validate=URL_INPUT_VALIDATION,
            key=key_prefix + "url",
            live=CUSTOM_MANGA_URL_LIVE,
            on_change=on_custom_manga_url_change,
            args=(key_prefix,),
        )

        run_pending_custom_manga_preset_detection(key_prefix)

        with st.expander(
            "Cover Image",
        ):
            st.text_input(
                "Cover Image URL",
                placeholder="https://example.com/image.jpg",
                type="url",
                validate=URL_INPUT_VALIDATION,
                key=key_prefix + "cover_img_url",
            )
            st.file_uploader(
                "Upload Cover Image",
                type=["png", "jpg", "jpeg"],
                key=key_prefix + "cover_img_file",
            )
            st.checkbox(
                "Use Mantium default cover image",
                key=key_prefix + "use_mantim_default_cover_img",
            )

        show_custom_manga_tracking_fields(
            key_prefix,
            "Get the last released chapter from the check URL, or else the manga URL, using the selectors above, without saving anything.",
        )

        update_clicked = st.button(
            "Update",
            key=key_prefix + "update",
            width="stretch",
            type="primary",
        )
        back_clicked = False
        if (back_multimanga_id := ss.get("update_custom_manga_back_to_multimanga_id")) is not None:
            back_clicked = st.button("Back", key=key_prefix + "back", width="stretch")
        # Fixed slot below the buttons, so the message doesn't move them and doesn't
        # stay after the next rerun
        message_slot = st.empty()
        if back_clicked:
            try:
                # Fetched again, so the Manage Mangas form shows the updated mangas
                multimanga = api_client.get_multimanga(back_multimanga_id)
            except Exception as ex:
                logger.exception(ex)
                message_slot.error(f"Error while getting multimanga: {get_api_error_detail(ex)}")
            else:
                ss["highlighted_custom_manga"] = None
                ss.pop("update_custom_manga_back_to_multimanga_id", None)
                ss["highlighted_multimanga"] = multimanga
                st.rerun()
        if update_clicked:
            with message_slot.container():
                try:
                    name = ss[key_prefix + "name"]
                    url = ss[key_prefix + "url"]
                    selectors = get_custom_manga_selector_values(key_prefix)

                    cover_img_url = ss[key_prefix + "cover_img_url"]
                    cover_img = (
                        ss[key_prefix + "cover_img_file"].getvalue()
                        if ss[key_prefix + "cover_img_file"]
                        else None
                    )
                    use_mantium_default_cover_img = ss[key_prefix + "use_mantim_default_cover_img"]

                    if name == "":
                        st.warning("Provide a manga name")
                    elif invalid_url_message := (
                        get_invalid_url_message("manga URL", url)
                        or get_invalid_url_message("cover image URL", cover_img_url)
                        or get_invalid_url_message("check URL", ss[key_prefix + "check_url"])
                    ):
                        st.warning(invalid_url_message)
                    else:
                        cover_img_values_count = sum(
                            [
                                bool(cover_img_url),
                                bool(cover_img),
                                use_mantium_default_cover_img,
                            ]
                        )
                        if cover_img_values_count > 1:
                            st.warning(
                                "To update the cover image, provide only one of: an image URL, an uploaded file, or the Mantium default cover image"
                            )
                        else:
                            with st.spinner("Updating manga..."):
                                api_client.update_custom_manga(
                                    manga["ID"],
                                    name,
                                    url,
                                    selectors["last_released_chapter_name_selector"],
                                    selectors["last_released_chapter_name_attribute"],
                                    selectors["last_released_chapter_name_regex"],
                                    selectors["last_released_chapter_name_get_first"],
                                    selectors["last_released_chapter_url_selector"],
                                    selectors["last_released_chapter_url_attribute"],
                                    selectors["last_released_chapter_url_get_first"],
                                    selectors["last_released_chapter_use_browser"],
                                    cover_img_url,
                                    cover_img if cover_img else b"",
                                    use_mantium_default_cover_img,
                                    check_url=ss[key_prefix + "check_url"],
                                )
                            ss["update_manga_success_message"] = (
                                "Manga updated successfully"
                            )
                            ss["highlighted_custom_manga"] = None
                            ss.pop("update_custom_manga_back_to_multimanga_id", None)
                            st.rerun()
                except Exception as ex:
                    logger.exception(ex)
                    if user_message := get_api_user_message(ex):
                        st.warning(user_message)
                    else:
                        st.error(f"Error while updating manga: {get_api_error_detail(ex)}")

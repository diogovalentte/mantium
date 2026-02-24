import logging
from streamlit import session_state as ss
import pathlib
from datetime import datetime, timezone
import uuid

import streamlit as st
from bs4 import BeautifulSoup
from dateutil.parser import parse
from streamlit_extras.stylable_container import stylable_container
from streamlit_javascript import st_javascript


def get_logger():
    logging.basicConfig(
        encoding="utf-8",
        level=logging.INFO,
        format="%(asctime)s :: %(levelname)-8s :: %(name)s :: %(message)s",
    )

    return logging.getLogger()


def get_updated_at_datetime(updated_at: str) -> datetime:
    if updated_at == "0001-01-01T00:00:00Z":
        return datetime.min.replace(tzinfo=timezone.utc)
    return parse(updated_at)


def remove_nano_from_datetime(datetime_string: str):
    if len(datetime_string) > 19:
        return datetime_string[:19] + "Z"
    else:
        return datetime_string


def centered_container(key: str):
    css_styles = """
        div {
            /* remove comment bellow to also center text */
            /* display: flex; */
            justify-content: center;
        }
        img {
            display: block;
            margin: auto;
        }
    """

    return stylable_container(key, css_styles)


def tagger(
    content: str,
    tag: str,
    background_color: str | None = "red",
    extra_css: str = "",
    text_color: str | None = "white",
):
    """Creates a tag element.

    Args:
        content (str): Content to be tagged.
        tag (str): Tag content, can be a simple text or HTML & CSS.
        background_color (str | None, optional): Tag background color. Can be any CSS valid color. Defaults to "red".
        extra_css (str, optional): Extra CSS to be added to the tag. Defaults to "".
        text_color (str | None, optional): Tag text color. Can be any CSS valid color. Defaults to "white".
    """
    html = f"""
        {content} <span style="display:inline-block;
        background-color: {background_color};
        padding: 0.1rem 0.5rem;
        font-size: 14px;
        font-weight: 400;
        color: {text_color};
        border-radius: 1rem;{extra_css}">{tag}</span>
    """

    st.write(html, unsafe_allow_html=True)


def get_relative_time(past_date):
    current_date = datetime.now()
    time_difference = current_date - past_date.replace(tzinfo=None)

    total_days = time_difference.days
    total_weeks = total_days // 7

    if total_weeks >= 4:
        return past_date.strftime("%Y-%m-%d")

    # Define the relative time format based on the difference
    if total_weeks >= 1:
        return f"{total_weeks} {'week' if total_weeks == 1 else 'weeks'} ago"
    elif total_days >= 2:
        return f"{total_days} {'day' if total_days == 1 else 'days'} ago"
    elif total_days == 1:
        return "Yesterday"
    elif time_difference.seconds >= 3600:  # 3600 seconds in an hour
        total_hours = time_difference.seconds // 3600
        return f"{total_hours} {'hour' if total_hours == 1 else 'hours'} ago"
    else:
        return "Just now"


def set_custom_js_to_none():
    js = """window.parent.document.querySelectorAll('div:has(> iframe[title="streamlit_javascript.streamlit_javascript"])').forEach(div => div.parentElement.style.display = 'none');"""
    st_javascript(js, key=str(uuid.uuid4()))


def fix_streamlit_index_html():
    """Fixes the Streamlit index.html file to allow to load mangadex images.

    DOING IT USING st_javascript IN THE MAIN FILE INSTEAD.
    """
    index_path = pathlib.Path(st.__file__).parent / "static" / "index.html"
    soup = BeautifulSoup(index_path.read_text(), features="html.parser")

    meta_tag = soup.find("meta", attrs={"name": "referrer", "content": "no-referrer"})
    if meta_tag:
        return

    head = soup.head

    meta_tag = soup.new_tag(
        "meta", attrs={"name": "referrer", "content": "no-referrer"}
    )

    head.insert(1, meta_tag)

    index_path.write_text(str(soup))

    return


def get_source_name_and_colors(source: str):
    """Returns the source name, text color, and background color.

    Args:
        source (str): Source.

    Returns:
        (str, str, str): Source name, text color, background color.
    """
    match source:
        case "mangadex":
            return "MangaDex", "white", "#ff6740"
        case "comick":
            return "ComicK", "white", "#1f2937"
        case "mangaplus":
            return "MangaPlus", "white", "#d40a15"
        case "mangahub":
            return "MangaHub", "white", "#dc98f1"
        case "mangaupdates":
            return "Manga Updates", "white", "#f69731"
        case "rawkuma":
            return "RawKuma", "white", "#0c70de"
        case "klmanga":
            return "KLManga", "white", "#ee2631"
        case "jmanga":
            return "JManga", "white", "#7b36ce"
        case _:
            return source, "black", "white"


def set_is_dialog_open():
    # This is used to prevent the dialog from closing when the user is interacting with it
    ss["is_dialog_open"] = False

import base64
import html
import json
import logging
from streamlit import session_state as ss
import pathlib
from datetime import datetime, timezone
from io import BytesIO
import uuid
from typing import Any
from urllib.parse import urlparse

import streamlit as st
from bs4 import BeautifulSoup
from dateutil.parser import parse
from PIL import Image
from streamlit_extras.stylable_container import stylable_container
from streamlit_javascript import st_javascript

import src.util.defaults as defaults


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


def get_api_user_message(ex: Exception) -> str | None:
    """Returns the error message the API marked as meant to be shown to the user, if any."""
    response_text = getattr(ex, "response_text", None)
    if not response_text:
        return None
    try:
        return json.loads(response_text).get("user_message") or None
    except (ValueError, AttributeError):
        return None


def get_api_error_detail(ex: Exception) -> str:
    """Returns the first line of the API error message, or of the exception if the API didn't answer."""
    response_text = getattr(ex, "response_text", None)
    message = None
    if response_text:
        try:
            message = json.loads(response_text).get("message")
        except (ValueError, AttributeError):
            message = response_text
    if not message:
        message = getattr(ex, "msg", None) or str(ex)

    return str(message).strip().splitlines()[0]


# Turns off the browser validation of st.text_input(type="url"). An invalid value isn't
# sent to the app, so the form would submit the previous valid one (or nothing) while
# showing the invalid one. The forms check the URL with get_invalid_url_message instead.
URL_INPUT_VALIDATION = ""


def get_invalid_url_message(field: str, url: str) -> str | None:
    """Returns a message if the URL is provided but isn't a valid http(s) URL."""
    if url == "":
        return None
    try:
        parsed = urlparse(url)
        hostname = parsed.hostname
    except ValueError:
        return f"Invalid {field}"
    if parsed.scheme not in ("http", "https") or not hostname:
        return f"Invalid {field}: must start with http:// or https://"
    return None


def get_url_hostname(url: str) -> str:
    """Returns the URL hostname, or an empty string if the URL is invalid."""
    try:
        return urlparse(url).hostname or ""
    except ValueError:
        return ""


def chapter_link(chapter: dict[str, Any], manga: dict[str, Any]) -> str:
    """Returns the link of a chapter. A chapter without URL, like a custom manga's chapter found only
    by its name, links to the manga. Empty if neither has a URL."""
    return chapter.get("URL") or manga.get("URL") or ""


def remove_nano_from_datetime(datetime_string: str):
    if len(datetime_string) > 19:
        return datetime_string[:19] + "Z"
    else:
        return datetime_string


def centered_container(key: str):
    css_styles = """
        div {
            /* remove comment below to also center text */
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
        case "custom_manga":
            return "Custom Manga", "black", "white"
        case _:
            return source, "black", "white"


def set_is_dialog_open():
    # This is used to prevent the dialog from closing when the user is interacting with it
    ss["is_dialog_open"] = False


def is_unread_chapter(last_read_chapter: dict[str, Any], last_released_chapter: dict[str, Any]) -> bool:
    """Returns True if the multimanga has unread chapters, False otherwise.

    The API uses the same rule for the library stats (manga.IsUnreadChapter), keep them in sync."""

    last_read_chapter_url = last_read_chapter.get("URL", "")
    last_released_chapter_url = last_released_chapter.get("URL", "")
    last_read_chapter = last_read_chapter["Chapter"]
    last_released_chapter = last_released_chapter["Chapter"]

    if last_released_chapter == "":
        return False
    elif last_read_chapter == last_released_chapter:
        # Every chapter of a custom manga tracked only by the chapter URL has the same name
        return (
            last_released_chapter == defaults.UNKNOWN_CHAPTER
            and last_read_chapter_url != last_released_chapter_url
        )

    try:
        last_read_chapter_number = float(last_read_chapter)
        last_released_chapter_number = float(last_released_chapter)
    except ValueError:
        return True

    return last_read_chapter_number < last_released_chapter_number


def escape_html(value) -> str:
    """Escape a value that is interpolated into an unsafe_allow_html block.

    Manga names reach the cards unescaped today, so a custom manga named
    "Teste <b>NEGRITO</b>" renders the tag as markup instead of showing it:
    the name displayed is not the name stored. Streamlit strips event
    handlers, so this is a correctness problem rather than a scripting one.
    """
    return html.escape(str(value), quote=True)


def web_url_or_none(url) -> str | None:
    """Returns the URL if it's an http(s) URL, for Streamlit link parameters, which escape it."""
    url = str(url).strip()
    if not url.lower().startswith(("http://", "https://")):
        return None
    return url


def safe_href(url) -> str:
    """Return url escaped for an href attribute, or "" if it is not a web URL.

    A custom manga's URL field is free text, so it can hold anything - only
    http and https should ever end up in a link.
    """
    url = str(url).strip()
    if not url.lower().startswith(("http://", "https://")):
        return ""

    return html.escape(url, quote=True)


def show_manga_cover(
    manga: dict[str, Any],
    size: tuple[int, int],
    always_resize: bool = False,
    cover_url_key: str = "CoverImgURL",
):
    """Shows the manga cover, linking to the manga page when it has one.

    The cover is the image bytes in manga["CoverImg"] when there are any, else the image at
    manga[cover_url_key], else Mantium's default cover.
    """
    link = web_url_or_none(manga["URL"])
    cover_img = manga.get("CoverImg")
    if cover_img:
        img = BytesIO(base64.b64decode(cover_img))
        if always_resize or not manga.get("CoverImgResized", False):
            img = Image.open(img).resize(size)
        st.image(img, link=link, alt=manga["Name"])
        return

    # st.image opens any string that isn't an http(s) or data URL as a local file, so a
    # relative cover URL from a source site would take the page down. It also can't give a
    # remote image a fixed size, which keeps the cards aligned, so these use an <img>.
    cover_url = safe_href(manga.get(cover_url_key) or "") or html.escape(defaults.DEFAULT_MANGA_COVER, quote=True)
    img_tag = f'<img src="{cover_url}" alt="{html.escape(str(manga["Name"]), quote=True)}" width="{size[0]}" height="{size[1]}" style="object-fit: cover;"/>'
    if link:
        img_tag = f'<a href="{html.escape(link, quote=True)}" target="_blank">{img_tag}</a>'
    st.markdown(img_tag, unsafe_allow_html=True)

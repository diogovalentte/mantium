"""App run by AppTest in test_update_custom_manga_form.py."""

import streamlit as st

import src.util.add_manga as add_manga
import src.util.update_manga as update_manga
from src.tests.src.util.fake_api import FAKE

add_manga.get_api_client = lambda: FAKE
update_manga.get_api_client = lambda: FAKE

# Tests set "manga" to None and back to simulate closing the dialog and opening it for a manga
if st.session_state.get("manga") is not None:
    update_manga.show_update_custom_manga(st.session_state["manga"])

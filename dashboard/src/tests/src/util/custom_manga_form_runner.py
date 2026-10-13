"""App run by AppTest in test_custom_manga_form.py. It can't be a function passed to
AppTest.from_function: that re-executes only the function source, without its imports."""

import streamlit as st

import src.util.add_manga as add_manga
from src.tests.src.util.fake_api import FAKE

add_manga.get_api_client = lambda: FAKE

# Tests set "show_form" to False and back to simulate closing and reopening the dialog
if st.session_state.get("show_form", True):
    add_manga.show_add_custom_manga_form(st.session_state.get("multimanga_id", 0))
if st.session_state.get("show_second_form", False):
    add_manga.show_add_custom_manga_form(7)

"""App run by AppTest in test_update_multimanga_form.py."""

import streamlit as st

import src.util.update_manga as update_manga
from src.tests.src.util.fake_api import FAKE

update_manga.get_api_client = lambda: FAKE

st.session_state.setdefault("is_mobile", False)
# Tests set "open_dialog" to run what runs when the dialog opens
if st.session_state.pop("open_dialog", False):
    update_manga.reset_update_multimanga_chapters()
update_manga.show_update_multimanga(1)

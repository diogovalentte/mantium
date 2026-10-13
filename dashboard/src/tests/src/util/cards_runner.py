"""Renders the manga cards of the dialogs (multimanga mangas and search results) for AppTest."""

import streamlit as st

import src.util.add_manga as add_manga
import src.util.update_manga as update_manga
from src.tests.src.util.fake_api import FAKE

add_manga.get_api_client = lambda: FAKE
update_manga.get_api_client = lambda: FAKE

manga = FAKE.get_mangas()[0]
update_manga.show_multimanga_manga(manga, 1, current_manga=True)
if "search_result" in st.session_state:
    add_manga.show_search_result_manga(st.session_state["search_result"], "Add", "picked_search_result")

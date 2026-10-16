"""Renders the Manage Mangas cards of a multimanga for AppTest in test_manage_mangas.py."""

import streamlit as st

import src.util.update_manga as update_manga
from src.tests.src.util.fake_api import FAKE

update_manga.get_api_client = lambda: FAKE

st.session_state.setdefault("is_mobile", False)
mangas = st.session_state["mangas"]
update_manga.show_multimanga_mangas(st.columns(2), mangas, mangas[0]["ID"], 1)

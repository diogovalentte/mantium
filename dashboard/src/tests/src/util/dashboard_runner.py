"""Runs the main dashboard page for AppTest, with the API, the browser detection and the
injected JavaScript replaced, since AppTest has neither a browser nor the API."""

import os
import runpy

import browser_detection
import streamlit as st
import streamlit_javascript

import src.api.api_client as api_client_module
import src.util.add_manga as add_manga
import src.util.update_manga as update_manga
import src.util.util as util
from src.tests.src.util.fake_api import FAKE

DASHBOARD = os.path.join(os.path.dirname(__file__), "..", "..", "..", "..", "01_📖_Dashboard.py")

browser_detection.browser_detection_engine = lambda *args, **kwargs: {
    "isMobile": st.session_state.get("fake_is_mobile", False)
}
streamlit_javascript.st_javascript = lambda *args, **kwargs: None
util.st_javascript = lambda *args, **kwargs: None
api_client_module.get_api_client = lambda: FAKE
add_manga.get_api_client = lambda: FAKE
update_manga.get_api_client = lambda: FAKE

runpy.run_path(DASHBOARD, run_name="__main__")

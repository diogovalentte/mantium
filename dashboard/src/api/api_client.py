import logging
import os

import streamlit as st
from src.api.manga_api import MangaAPIClient
from src.api.multimanga_api import MultiMangaAPIClient
from src.api.system_api import DashboardAPIClient
from streamlit import session_state as ss


@st.cache_data()
def get_api_client():
    logger = logging.getLogger("api_client")
    logger.info("Defining the API client...")

    api_address = os.environ.get("API_ADDRESS", "")
    if api_address == "":
        api_address = "http://localhost:8080"
    logger.info(f"API address: {api_address}")

    api_client = APIClient(api_address)

    logger.info("API client defined")

    return api_client


class APIClient(MangaAPIClient, MultiMangaAPIClient, DashboardAPIClient):
    def __init__(self, base_url: str) -> None:
        self.base_api_url = base_url
        super().__init__(self.base_api_url)
        MultiMangaAPIClient.__init__(self, self.base_api_url)
        DashboardAPIClient.__init__(self, self.base_api_url)

    @st.cache_data(show_spinner=False, max_entries=5, ttl=600)
    def get_cached_manga_chapters(_, id: int, url: str, internal_id: str):
        api_client = get_api_client()
        chapters = api_client.get_manga_chapters(id, url, internal_id)

        return chapters

    @st.cache_data(show_spinner=False, max_entries=5, ttl=600)
    def get_cached_multimanga_chapters(_, id: int, manga_id: int):
        api_client = get_api_client()
        chapters = api_client.get_multimanga_chapters(id, manga_id)

        return chapters

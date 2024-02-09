import logging

import streamlit as st
from src.api.manga_api import MangaAPIClient
from src.api.system_api import SystemAPIClient


@st.cache_data()
def get_api_client():
    api_client = st.session_state.get("api_client", None)
    if api_client is None:
        logger = logging.getLogger("api_client")
        logger.info("Defining the API client...")

        api_client = APIClient(
            "http://mangas-dashboard-api", 8080
        )  # The golang API docker service name
        st.session_state["api_client"] = api_client

        logger.info("API client defined")

    return api_client


class APIClient(MangaAPIClient, SystemAPIClient):
    def __init__(self, base_URL: str, port: int) -> None:
        self.base_url = f"{base_URL}:{port}"
        super().__init__(self.base_url)

import logging
import os

import streamlit as st
from src.api.manga_api import MangaAPIClient
from src.api.system_api import DashboardAPIClient


@st.cache_data()
def get_api_client():
    api_client = st.session_state.get("api_client", None)
    if api_client is None:
        logger = logging.getLogger("api_client")
        logger.info("Defining the API client...")

        api_address = os.environ.get("API_ADDRESS", "")
        if api_address == "":
            raise ValueError("API_ADDRESS environment variable is not set")

        api_client = APIClient(api_address)  # The golang API docker service name
        st.session_state["api_client"] = api_client

        logger.info("API client defined")

    return api_client


class APIClient(MangaAPIClient, DashboardAPIClient):
    def __init__(self, base_url: str) -> None:
        self.base_url = base_url
        super().__init__(self.base_url)

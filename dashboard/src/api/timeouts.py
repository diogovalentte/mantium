"""Timeouts for every request the dashboard makes to the Mantium API.

`requests` has no timeout by default. If the API accepts the connection and
then never answers, the call blocks forever and takes the Streamlit worker
thread with it - the tab stops responding and never recovers, including the
five second polling fragment that checks for dashboard updates.

Both values are (connect, read) pairs in seconds.
"""

# Endpoints that only read Mantium's own state and should answer immediately.
REQUEST_TIMEOUT = (5, 30)

# Endpoints that reach out to a manga source (searching, fetching metadata and
# chapters, adding a manga, downloading a cover). These can legitimately take a
# while, especially when the source needs the headless browser.
SOURCE_REQUEST_TIMEOUT = (5, 300)

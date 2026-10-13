import pytest
import streamlit.elements.lib.policies as policies


@pytest.fixture(autouse=True)
def widget_state_duplication_warnings(monkeypatch):
    """Collects the "created with a default value but also had its value set via the
    Session State API" warnings. Since Streamlit 1.56 they're only logged, once per
    process, so tests can't see them in the app."""
    messages = []

    class Logger:
        def warning(self, message, *args, **kwargs):
            messages.append(message % args)

        def __getattr__(self, name):
            return getattr(policies._LOGGER.__class__, name)

    monkeypatch.setattr(policies, "_shown_default_value_warning", False)
    monkeypatch.setattr(policies, "_LOGGER", Logger())
    yield messages
    assert not messages, messages

from datetime import datetime, timedelta

from src.util import get_relative_time

test_get_relative_time_table = [
    {
        "past_date": datetime.now() - timedelta(hours=1),
        "expected": "1 hour ago",
    },
    {
        "past_date": datetime.now() - timedelta(hours=2),
        "expected": "2 hours ago",
    },
    {
        "past_date": datetime.now() - timedelta(hours=24),
        "expected": "Yesterday",
    },
    {
        "past_date": datetime.now() - timedelta(days=1),
        "expected": "Yesterday",
    },
    {
        "past_date": datetime.now() - timedelta(days=4),
        "expected": "4 days ago",
    },
    {
        "past_date": datetime.now() - timedelta(days=7),
        "expected": "1 week ago",
    },
    {
        "past_date": datetime.now() - timedelta(weeks=1),
        "expected": "1 week ago",
    },
    {
        "past_date": datetime.now() - timedelta(weeks=3),
        "expected": "3 weeks ago",
    },
    {
        "past_date": datetime.now() - timedelta(weeks=4),
        "expected": (datetime.today() - timedelta(weeks=4)).date().strftime("%Y-%m-%d"),
    },
    {
        "past_date": datetime(2021, 1, 1),
        "expected": "2021-01-01",
    },
    {
        "past_date": datetime(2021, 12, 31),
        "expected": "2021-12-31",
    },
]


def test_get_relative_time():
    for test in test_get_relative_time_table:
        past_date = test["past_date"]
        result = get_relative_time(past_date)

        assert result == test["expected"]

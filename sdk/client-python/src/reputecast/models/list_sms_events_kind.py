from enum import StrEnum

class ListSmsEventsKind(StrEnum):
    ALL = "all"
    FAILURES = "failures"
    KEYWORDS = "keywords"
    RECEIPTS = "receipts"
    REPLIES = "replies"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class ListRepliesView(StrEnum):
    AUTO = "auto"
    DONE = "done"
    OPEN = "open"
    OTHER = "other"
    SPAM = "spam"
    UNSUBSCRIBES = "unsubscribes"

    def __str__(self) -> str:
        return str(self.value)

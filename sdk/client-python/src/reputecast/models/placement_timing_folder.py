from enum import StrEnum

class PlacementTimingFolder(StrEnum):
    INBOX = "inbox"
    SPAM = "spam"
    TAB = "tab"

    def __str__(self) -> str:
        return str(self.value)

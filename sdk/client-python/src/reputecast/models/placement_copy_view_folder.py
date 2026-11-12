from enum import StrEnum

class PlacementCopyViewFolder(StrEnum):
    INBOX = "inbox"
    SPAM = "spam"
    TAB = "tab"
    VALUE_3 = ""

    def __str__(self) -> str:
        return str(self.value)

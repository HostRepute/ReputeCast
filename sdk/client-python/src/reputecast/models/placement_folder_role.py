from enum import StrEnum

class PlacementFolderRole(StrEnum):
    INBOX = "inbox"
    SPAM = "spam"
    TAB = "tab"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class LiveCancelRequestMailbox(StrEnum):
    GMAIL = "gmail"
    ICLOUD = "icloud"
    OTHER = "other"
    OUTLOOK = "outlook"
    YAHOO = "yahoo"

    def __str__(self) -> str:
        return str(self.value)

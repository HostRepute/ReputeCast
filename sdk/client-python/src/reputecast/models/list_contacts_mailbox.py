from enum import StrEnum

class ListContactsMailbox(StrEnum):
    GMAIL = "gmail"
    ICLOUD = "icloud"
    OTHER = "other"
    OUTLOOK = "outlook"
    VALUE_5 = ""
    YAHOO = "yahoo"

    def __str__(self) -> str:
        return str(self.value)

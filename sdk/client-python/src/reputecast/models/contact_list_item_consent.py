from enum import StrEnum

class ContactListItemConsent(StrEnum):
    DOUBLE = "double"
    PENDING = "pending"
    SINGLE = "single"

    def __str__(self) -> str:
        return str(self.value)

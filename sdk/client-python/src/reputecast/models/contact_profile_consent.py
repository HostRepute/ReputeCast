from enum import StrEnum

class ContactProfileConsent(StrEnum):
    DOUBLE = "double"
    PENDING = "pending"
    SINGLE = "single"

    def __str__(self) -> str:
        return str(self.value)

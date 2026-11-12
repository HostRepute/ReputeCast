from enum import StrEnum

class ContactFilterConsent(StrEnum):
    DOUBLE = "double"
    PENDING = "pending"
    SINGLE = "single"

    def __str__(self) -> str:
        return str(self.value)

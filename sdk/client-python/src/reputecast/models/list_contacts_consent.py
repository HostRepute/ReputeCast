from enum import StrEnum

class ListContactsConsent(StrEnum):
    DOUBLE = "double"
    PENDING = "pending"
    SINGLE = "single"
    VALUE_3 = ""

    def __str__(self) -> str:
        return str(self.value)

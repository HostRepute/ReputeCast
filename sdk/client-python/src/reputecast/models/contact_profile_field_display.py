from enum import StrEnum

class ContactProfileFieldDisplay(StrEnum):
    CURRENCY = "currency"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)

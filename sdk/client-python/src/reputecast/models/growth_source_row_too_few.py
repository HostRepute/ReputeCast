from enum import StrEnum

class GrowthSourceRowTooFew(StrEnum):
    CONTACTS = "contacts"
    MESSAGES = "messages"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)

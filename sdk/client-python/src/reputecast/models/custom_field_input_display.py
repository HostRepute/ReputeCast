from enum import StrEnum

class CustomFieldInputDisplay(StrEnum):
    CURRENCY = "currency"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)

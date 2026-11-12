from enum import StrEnum

class NewContactConsent(StrEnum):
    DOUBLE = "double"
    SINGLE = "single"

    def __str__(self) -> str:
        return str(self.value)

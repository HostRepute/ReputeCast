from enum import StrEnum

class ConversionResultMatched(StrEnum):
    CLICK_TOKEN = "click_token"
    EMAIL = "email"
    NONE = "none"

    def __str__(self) -> str:
        return str(self.value)

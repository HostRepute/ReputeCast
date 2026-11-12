from enum import StrEnum

class CommerceActivityItemMatched(StrEnum):
    CLICK_TOKEN = "click_token"
    EMAIL = "email"
    NONE = "none"
    VALUE_3 = ""

    def __str__(self) -> str:
        return str(self.value)

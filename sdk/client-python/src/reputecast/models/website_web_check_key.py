from enum import StrEnum

class WebsiteWebCheckKey(StrEnum):
    ADDRESS = "address"
    COVER = "cover"
    HIDDEN = "hidden"
    TOKENS = "tokens"

    def __str__(self) -> str:
        return str(self.value)

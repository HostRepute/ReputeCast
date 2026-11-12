from enum import StrEnum

class ContentCheckLinkKind(StrEnum):
    BUTTON = "button"
    IMAGE = "image"
    LINK = "link"
    NAV = "nav"
    SOCIAL = "social"

    def __str__(self) -> str:
        return str(self.value)

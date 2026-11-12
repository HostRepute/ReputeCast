from enum import StrEnum

class MJMLThumbItemKind(StrEnum):
    BUTTON = "button"
    FOOTER = "footer"
    HEADER = "header"
    HEADING = "heading"
    IMAGE = "image"
    TEXT = "text"

    def __str__(self) -> str:
        return str(self.value)

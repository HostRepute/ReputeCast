from enum import StrEnum

class AccessibilityFixSpecKind(StrEnum):
    ALT_TEXT = "alt_text"
    COLOR = "color"
    HEADING = "heading"
    LINK_TEXT = "link_text"
    TEXT_SIZE = "text_size"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class AccessibilityFixInputKind(StrEnum):
    ALT_TEXT = "alt_text"
    COLOR = "color"
    DECORATIVE = "decorative"
    HEADING = "heading"
    LINK_TEXT = "link_text"
    TEXT_ROLE = "text_role"
    TEXT_SIZE = "text_size"

    def __str__(self) -> str:
        return str(self.value)

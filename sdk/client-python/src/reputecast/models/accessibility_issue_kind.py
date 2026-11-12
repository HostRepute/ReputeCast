from enum import StrEnum

class AccessibilityIssueKind(StrEnum):
    ALT_TEXT = "alt_text"
    CONTRAST = "contrast"
    HEADING = "heading"
    LINK_NO_TEXT = "link_no_text"
    LINK_TEXT = "link_text"
    TEXT_SIZE = "text_size"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class AccessibilityFixInputTarget(StrEnum):
    BACKGROUND = "background"
    TEXT = "text"

    def __str__(self) -> str:
        return str(self.value)

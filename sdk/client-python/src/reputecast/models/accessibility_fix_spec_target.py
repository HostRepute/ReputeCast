from enum import StrEnum

class AccessibilityFixSpecTarget(StrEnum):
    BACKGROUND = "background"
    TEXT = "text"

    def __str__(self) -> str:
        return str(self.value)

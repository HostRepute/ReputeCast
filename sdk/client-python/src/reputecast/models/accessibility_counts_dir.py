from enum import StrEnum

class AccessibilityCountsDir(StrEnum):
    LTR = "ltr"

    def __str__(self) -> str:
        return str(self.value)

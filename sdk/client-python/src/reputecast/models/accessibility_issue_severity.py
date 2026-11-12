from enum import StrEnum

class AccessibilityIssueSeverity(StrEnum):
    MODERATE = "moderate"
    SERIOUS = "serious"

    def __str__(self) -> str:
        return str(self.value)

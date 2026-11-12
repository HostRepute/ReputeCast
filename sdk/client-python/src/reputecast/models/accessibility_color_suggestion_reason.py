from enum import StrEnum

class AccessibilityColorSuggestionReason(StrEnum):
    AAA = "aaa"
    NEAREST_AA = "nearest_aa"
    OTHER_BUTTONS = "other_buttons"

    def __str__(self) -> str:
        return str(self.value)

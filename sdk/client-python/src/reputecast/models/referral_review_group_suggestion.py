from enum import StrEnum

class ReferralReviewGroupSuggestion(StrEnum):
    COUNT = "count"
    DONT_PAUSE = "dont_pause"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)

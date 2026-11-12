from enum import StrEnum

class ReferralDecisionInputDecision(StrEnum):
    COUNT = "count"
    DONT = "dont"
    DONT_PAUSE = "dont_pause"

    def __str__(self) -> str:
        return str(self.value)

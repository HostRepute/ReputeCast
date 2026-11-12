from enum import StrEnum

class ScoringRecalcInfoReason(StrEnum):
    RETRY = "retry"
    RULES = "rules"
    SETUP = "setup"

    def __str__(self) -> str:
        return str(self.value)

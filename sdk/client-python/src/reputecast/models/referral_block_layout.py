from enum import StrEnum

class ReferralBlockLayout(StrEnum):
    LINE = "line"
    PROGRESS = "progress"
    REWARDS = "rewards"

    def __str__(self) -> str:
        return str(self.value)

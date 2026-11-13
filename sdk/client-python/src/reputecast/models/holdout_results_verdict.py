from enum import StrEnum

class HoldoutResultsVerdict(StrEnum):
    ADDED = "added"
    LOST = "lost"
    NONE = "none"
    NO_DIFFERENCE = "no_difference"
    PENDING = "pending"
    TOO_EARLY = "too_early"
    TOO_FEW = "too_few"

    def __str__(self) -> str:
        return str(self.value)

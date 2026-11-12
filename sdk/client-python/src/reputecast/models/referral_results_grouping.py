from enum import StrEnum

class ReferralResultsGrouping(StrEnum):
    DAY = "day"
    WEEK = "week"

    def __str__(self) -> str:
        return str(self.value)

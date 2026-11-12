from enum import StrEnum

class ReferralLeaderboardSettingsPeriodsItem(StrEnum):
    ALL = "all"
    MONTH = "month"

    def __str__(self) -> str:
        return str(self.value)

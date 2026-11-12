from enum import IntEnum

class ReferralLeaderboardSettingsRows(IntEnum):
    VALUE_5 = 5
    VALUE_10 = 10
    VALUE_25 = 25

    def __str__(self) -> str:
        return str(self.value)

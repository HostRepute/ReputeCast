from enum import StrEnum

class ReferralResultsPeriod(StrEnum):
    ALL = "all"
    VALUE_0 = "7d"
    VALUE_1 = "30d"

    def __str__(self) -> str:
        return str(self.value)

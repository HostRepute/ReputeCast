from enum import IntEnum

class ReferralCountingHoldDays(IntEnum):
    VALUE_0 = 0
    VALUE_3 = 3
    VALUE_7 = 7
    VALUE_14 = 14
    VALUE_30 = 30

    def __str__(self) -> str:
        return str(self.value)

from enum import IntEnum

class TrafficThresholdsDeferralWindow(IntEnum):
    VALUE_5 = 5
    VALUE_10 = 10
    VALUE_15 = 15
    VALUE_30 = 30

    def __str__(self) -> str:
        return str(self.value)

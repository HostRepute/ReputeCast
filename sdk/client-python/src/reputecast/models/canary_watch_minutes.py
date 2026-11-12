from enum import IntEnum

class CanaryWatchMinutes(IntEnum):
    VALUE_10 = 10
    VALUE_20 = 20
    VALUE_30 = 30
    VALUE_60 = 60

    def __str__(self) -> str:
        return str(self.value)

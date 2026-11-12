from enum import IntEnum

class WarmupSettingsDays(IntEnum):
    VALUE_14 = 14
    VALUE_21 = 21
    VALUE_30 = 30

    def __str__(self) -> str:
        return str(self.value)

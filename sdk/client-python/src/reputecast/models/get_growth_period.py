from enum import StrEnum

class GetGrowthPeriod(StrEnum):
    CUSTOM = "custom"
    VALUE_0 = "30d"
    VALUE_1 = "90d"
    VALUE_2 = "12m"
    VALUE_4 = ""

    def __str__(self) -> str:
        return str(self.value)

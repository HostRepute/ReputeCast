from enum import StrEnum

class DigestScheduleMonthRule(StrEnum):
    DAY = "day"
    FIRST = "first"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)

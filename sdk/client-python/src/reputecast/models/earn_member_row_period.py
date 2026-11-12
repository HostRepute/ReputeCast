from enum import StrEnum

class EarnMemberRowPeriod(StrEnum):
    MONTH = "month"
    VALUE_2 = ""
    YEAR = "year"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class ReportComparisonUnit(StrEnum):
    PERCENT = "percent"
    POINTS = "points"

    def __str__(self) -> str:
        return str(self.value)

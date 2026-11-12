from enum import StrEnum

class ReportABTestMetric(StrEnum):
    CLICKS = "clicks"
    HUMAN_OPENS = "human_opens"

    def __str__(self) -> str:
        return str(self.value)

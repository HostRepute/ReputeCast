from enum import StrEnum

class ReportABTestOutcome(StrEnum):
    MANUAL = "manual"
    NOT_SIGNIFICANT = "not_significant"
    TOO_SMALL = "too_small"
    UNRECORDED = "unrecorded"
    VALUE_5 = ""
    WINNER = "winner"

    def __str__(self) -> str:
        return str(self.value)

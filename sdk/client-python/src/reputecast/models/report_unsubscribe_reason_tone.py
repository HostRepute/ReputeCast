from enum import StrEnum

class ReportUnsubscribeReasonTone(StrEnum):
    BAD = "bad"
    INK = "ink"
    MUTED = "muted"

    def __str__(self) -> str:
        return str(self.value)

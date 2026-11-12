from enum import StrEnum

class ReportBounceRowSuppression(StrEnum):
    ADD = "add"
    AUTO = "auto"
    INVESTIGATE = "investigate"
    RETRY = "retry"
    REVIEW = "review"
    SUPPRESSED = "suppressed"

    def __str__(self) -> str:
        return str(self.value)

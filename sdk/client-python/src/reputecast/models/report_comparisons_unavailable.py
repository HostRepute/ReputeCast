from enum import StrEnum

class ReportComparisonsUnavailable(StrEnum):
    NOT_FINAL = "not_final"
    NO_HISTORY = "no_history"

    def __str__(self) -> str:
        return str(self.value)

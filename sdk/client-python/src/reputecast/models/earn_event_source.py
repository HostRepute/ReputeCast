from enum import StrEnum

class EarnEventSource(StrEnum):
    GAP_CHECK = "gap_check"
    HISTORY = "history"
    MANUAL = "manual"
    WEBHOOK = "webhook"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class SmsReportPhase(StrEnum):
    CANCELLED = "cancelled"
    COMPLETED = "completed"
    PAUSED = "paused"
    SCHEDULED = "scheduled"
    SENDING = "sending"
    VALUE_5 = ""

    def __str__(self) -> str:
        return str(self.value)

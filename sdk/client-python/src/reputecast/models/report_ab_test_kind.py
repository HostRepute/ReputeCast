from enum import StrEnum

class ReportABTestKind(StrEnum):
    CONTENT = "content"
    SEND_TIME = "send_time"
    SUBJECT = "subject"

    def __str__(self) -> str:
        return str(self.value)

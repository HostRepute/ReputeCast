from enum import StrEnum

class SmsReportScheduleMode(StrEnum):
    AT = "at"
    LOCAL = "local"
    NOW = "now"

    def __str__(self) -> str:
        return str(self.value)

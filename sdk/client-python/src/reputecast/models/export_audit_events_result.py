from enum import StrEnum

class ExportAuditEventsResult(StrEnum):
    ALLOWED = "allowed"
    DENIED = "denied"
    ERROR = "error"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class ListAuditEventsResult(StrEnum):
    ALLOWED = "allowed"
    DENIED = "denied"
    ERROR = "error"

    def __str__(self) -> str:
        return str(self.value)

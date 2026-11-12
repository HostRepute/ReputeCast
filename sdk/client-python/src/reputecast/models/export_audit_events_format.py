from enum import StrEnum

class ExportAuditEventsFormat(StrEnum):
    CSV = "csv"
    JSON = "json"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class ExportAuditEventsRange(StrEnum):
    ALL = "all"
    VALUE_0 = "24h"
    VALUE_1 = "7d"
    VALUE_2 = "30d"
    VALUE_3 = "90d"

    def __str__(self) -> str:
        return str(self.value)

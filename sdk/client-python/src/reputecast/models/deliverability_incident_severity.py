from enum import StrEnum

class DeliverabilityIncidentSeverity(StrEnum):
    BAD = "bad"
    INFO = "info"
    OK = "ok"
    WARN = "warn"

    def __str__(self) -> str:
        return str(self.value)

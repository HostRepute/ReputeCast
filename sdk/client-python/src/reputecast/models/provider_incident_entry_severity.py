from enum import StrEnum

class ProviderIncidentEntrySeverity(StrEnum):
    BAD = "bad"
    INFO = "info"
    OK = "ok"
    WARN = "warn"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class ProviderTrackingCheckStatus(StrEnum):
    OK = "ok"
    UNKNOWN = "unknown"
    WARN = "warn"

    def __str__(self) -> str:
        return str(self.value)

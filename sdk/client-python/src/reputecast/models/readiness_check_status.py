from enum import StrEnum

class ReadinessCheckStatus(StrEnum):
    BLOCK = "block"
    PASS = "pass"
    UNKNOWN = "unknown"
    WARN = "warn"

    def __str__(self) -> str:
        return str(self.value)

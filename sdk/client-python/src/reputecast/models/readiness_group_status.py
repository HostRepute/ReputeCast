from enum import StrEnum

class ReadinessGroupStatus(StrEnum):
    BLOCK = "block"
    PASS = "pass"
    UNKNOWN = "unknown"
    WARN = "warn"

    def __str__(self) -> str:
        return str(self.value)

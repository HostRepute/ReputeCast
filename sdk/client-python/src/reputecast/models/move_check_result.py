from enum import StrEnum

class MoveCheckResult(StrEnum):
    PASS = "pass"
    UNKNOWN = "unknown"
    WARN = "warn"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class ReadinessProgressGroupStatus(StrEnum):
    BLOCK = "block"
    PASS = "pass"
    QUEUED = "queued"
    RUNNING = "running"
    UNKNOWN = "unknown"
    WARN = "warn"

    def __str__(self) -> str:
        return str(self.value)

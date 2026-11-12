from enum import StrEnum

class CheckLineStatus(StrEnum):
    BAD = "bad"
    OK = "ok"
    SKIPPED = "skipped"
    WARN = "warn"

    def __str__(self) -> str:
        return str(self.value)

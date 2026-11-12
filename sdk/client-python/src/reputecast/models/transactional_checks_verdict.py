from enum import StrEnum

class TransactionalChecksVerdict(StrEnum):
    BLOCKED = "blocked"
    READY = "ready"
    WARNINGS = "warnings"

    def __str__(self) -> str:
        return str(self.value)

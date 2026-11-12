from enum import StrEnum

class ApprovalReadinessViewVerdict(StrEnum):
    BLOCKED = "blocked"
    READY = "ready"
    WARNINGS = "warnings"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class OverviewReadinessIssueStatus(StrEnum):
    BLOCK = "block"
    WARN = "warn"

    def __str__(self) -> str:
        return str(self.value)

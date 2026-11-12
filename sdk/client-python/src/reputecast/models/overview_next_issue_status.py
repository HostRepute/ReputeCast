from enum import StrEnum

class OverviewNextIssueStatus(StrEnum):
    COLLECTING = "collecting"
    DRAFT = "draft"
    HELD = "held"
    PREPARED = "prepared"
    PREPARING = "preparing"
    WAITING = "waiting"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class AutomationListItemStatus(StrEnum):
    DRAFT = "draft"
    LIVE = "live"
    PAUSED = "paused"

    def __str__(self) -> str:
        return str(self.value)

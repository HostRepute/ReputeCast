from enum import StrEnum

class ListAutomationsStatus(StrEnum):
    ALL = "all"
    DRAFTS = "drafts"
    LIVE = "live"
    PAUSED = "paused"

    def __str__(self) -> str:
        return str(self.value)

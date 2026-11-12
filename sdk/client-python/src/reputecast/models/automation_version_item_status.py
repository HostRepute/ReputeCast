from enum import StrEnum

class AutomationVersionItemStatus(StrEnum):
    DRAFT = "draft"
    LIVE = "live"
    RETIRED = "retired"
    RETIRING = "retiring"

    def __str__(self) -> str:
        return str(self.value)

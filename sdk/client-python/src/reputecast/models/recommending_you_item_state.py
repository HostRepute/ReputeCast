from enum import StrEnum

class RecommendingYouItemState(StrEnum):
    CONNECTED = "connected"
    INVALID = "invalid"
    PARTNER_PAUSED = "partner_paused"
    PAUSED = "paused"

    def __str__(self) -> str:
        return str(self.value)

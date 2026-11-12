from enum import StrEnum

class RecommendationItemState(StrEnum):
    CONNECTED = "connected"
    ENDED = "ended"
    INVALID = "invalid"
    LIMIT = "limit"
    PARTNER_PAUSED = "partner_paused"
    PAUSED = "paused"
    SHOWING = "showing"
    UNREACHABLE = "unreachable"

    def __str__(self) -> str:
        return str(self.value)

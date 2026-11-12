from enum import StrEnum

class OfferState(StrEnum):
    DRAFT = "draft"
    ENDED = "ended"
    LIMIT = "limit"
    LIVE = "live"
    MISSING = "missing"

    def __str__(self) -> str:
        return str(self.value)

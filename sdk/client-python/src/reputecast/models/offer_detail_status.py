from enum import StrEnum

class OfferDetailStatus(StrEnum):
    ENDED = "ended"
    LIVE = "live"

    def __str__(self) -> str:
        return str(self.value)

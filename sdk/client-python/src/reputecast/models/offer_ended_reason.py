from enum import StrEnum

class OfferEndedReason(StrEnum):
    DATE = "date"
    EARLY = "early"
    LIMIT = "limit"

    def __str__(self) -> str:
        return str(self.value)

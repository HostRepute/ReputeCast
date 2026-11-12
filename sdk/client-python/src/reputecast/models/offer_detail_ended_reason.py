from enum import StrEnum

class OfferDetailEndedReason(StrEnum):
    DATE = "date"
    EARLY = "early"
    LIMIT = "limit"

    def __str__(self) -> str:
        return str(self.value)

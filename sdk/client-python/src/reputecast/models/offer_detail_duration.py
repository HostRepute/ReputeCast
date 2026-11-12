from enum import StrEnum

class OfferDetailDuration(StrEnum):
    FOREVER = "forever"
    MONTHS = "months"
    ONCE = "once"

    def __str__(self) -> str:
        return str(self.value)

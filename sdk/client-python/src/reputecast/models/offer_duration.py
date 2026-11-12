from enum import StrEnum

class OfferDuration(StrEnum):
    FOREVER = "forever"
    MONTHS = "months"
    ONCE = "once"

    def __str__(self) -> str:
        return str(self.value)

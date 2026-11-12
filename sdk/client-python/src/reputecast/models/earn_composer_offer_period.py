from enum import StrEnum

class EarnComposerOfferPeriod(StrEnum):
    MONTH = "month"
    YEAR = "year"

    def __str__(self) -> str:
        return str(self.value)

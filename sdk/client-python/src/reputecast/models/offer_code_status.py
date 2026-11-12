from enum import StrEnum

class OfferCodeStatus(StrEnum):
    EXPIRED = "expired"
    NOT_REDEEMED = "not_redeemed"
    REDEEMED = "redeemed"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class OfferDetailDiscountType(StrEnum):
    AMOUNT = "amount"
    PERCENT = "percent"

    def __str__(self) -> str:
        return str(self.value)

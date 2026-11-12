from enum import StrEnum

class OfferDiscountType(StrEnum):
    AMOUNT = "amount"
    PERCENT = "percent"

    def __str__(self) -> str:
        return str(self.value)

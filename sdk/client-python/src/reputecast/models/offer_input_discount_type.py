from enum import StrEnum

class OfferInputDiscountType(StrEnum):
    AMOUNT = "amount"
    PERCENT = "percent"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class EarnMemberRowKind(StrEnum):
    COMPLIMENTARY = "complimentary"
    GIFT = "gift"
    STRIPE = "stripe"

    def __str__(self) -> str:
        return str(self.value)

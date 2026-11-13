from enum import StrEnum

class EarnMemberKind(StrEnum):
    COMPLIMENTARY = "complimentary"
    GIFT = "gift"
    GROUP = "group"
    STRIPE = "stripe"

    def __str__(self) -> str:
        return str(self.value)

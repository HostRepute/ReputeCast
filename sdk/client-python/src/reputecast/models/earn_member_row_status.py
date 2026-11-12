from enum import StrEnum

class EarnMemberRowStatus(StrEnum):
    ACTIVE = "active"
    CANCELED = "canceled"
    COMPLIMENTARY = "complimentary"
    GIFT = "gift"
    GROUP = "group"
    PAST_DUE = "past_due"
    TRIALING = "trialing"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class ContactPaidStatus(StrEnum):
    ACTIVE = "active"
    COMPLIMENTARY = "complimentary"
    ENDED = "ended"
    GIFT = "gift"
    PAST_DUE = "past_due"
    TRIALING = "trialing"

    def __str__(self) -> str:
        return str(self.value)

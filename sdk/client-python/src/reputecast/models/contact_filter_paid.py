from enum import StrEnum

class ContactFilterPaid(StrEnum):
    COMPLIMENTARY = "complimentary"
    ENDED = "ended"
    FREE = "free"
    GIFT = "gift"
    PAST_DUE = "past_due"
    TRIALING = "trialing"
    WITH_ACCESS = "with_access"

    def __str__(self) -> str:
        return str(self.value)

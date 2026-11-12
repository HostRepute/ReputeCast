from enum import StrEnum

class ListContactsPaid(StrEnum):
    COMPLIMENTARY = "complimentary"
    ENDED = "ended"
    FREE = "free"
    GIFT = "gift"
    PAST_DUE = "past_due"
    TRIALING = "trialing"
    VALUE_7 = ""
    WITH_ACCESS = "with_access"

    def __str__(self) -> str:
        return str(self.value)

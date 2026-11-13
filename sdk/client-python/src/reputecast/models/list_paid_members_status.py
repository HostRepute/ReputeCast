from enum import StrEnum

class ListPaidMembersStatus(StrEnum):
    CANCELS_SOON = "cancels_soon"
    COMPLIMENTARY = "complimentary"
    ENDED = "ended"
    GIFT = "gift"
    PAST_DUE = "past_due"
    PAUSED = "paused"
    PAYING = "paying"
    TRIALING = "trialing"
    WITH_ACCESS = "with_access"

    def __str__(self) -> str:
        return str(self.value)

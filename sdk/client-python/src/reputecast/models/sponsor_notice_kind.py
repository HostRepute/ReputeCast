from enum import StrEnum

class SponsorNoticeKind(StrEnum):
    CREATIVE_LATE = "creative_late"
    HOLD_LAPSING = "hold_lapsing"
    NO_CAMPAIGN = "no_campaign"
    PAYMENT_OVERDUE = "payment_overdue"

    def __str__(self) -> str:
        return str(self.value)

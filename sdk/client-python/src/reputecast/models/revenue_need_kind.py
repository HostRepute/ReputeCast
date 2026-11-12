from enum import StrEnum

class RevenueNeedKind(StrEnum):
    CREATIVE = "creative"
    MEMBER_EMAIL = "member_email"
    OFFER = "offer"
    PAST_DUE = "past_due"
    SPONSOR_PAYMENT = "sponsor_payment"

    def __str__(self) -> str:
        return str(self.value)

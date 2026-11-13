from enum import StrEnum

class SponsorRequestDetailStatus(StrEnum):
    AWAITING_PAYMENT = "awaiting_payment"
    BOOKED = "booked"
    CANCELED = "canceled"
    DECLINED = "declined"
    EXPIRED = "expired"
    REQUESTED = "requested"

    def __str__(self) -> str:
        return str(self.value)

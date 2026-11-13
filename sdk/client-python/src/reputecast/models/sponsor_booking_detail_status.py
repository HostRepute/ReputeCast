from enum import StrEnum

class SponsorBookingDetailStatus(StrEnum):
    AWAITING_PAYMENT = "awaiting_payment"
    BOOKED = "booked"
    CANCELED = "canceled"
    DECLINED = "declined"
    EXPIRED = "expired"
    HOLD = "hold"
    PROPOSED = "proposed"
    REQUESTED = "requested"
    SENT = "sent"

    def __str__(self) -> str:
        return str(self.value)

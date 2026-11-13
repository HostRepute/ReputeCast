from enum import StrEnum

class SponsorBookingBriefStatus(StrEnum):
    AWAITING_PAYMENT = "awaiting_payment"
    BOOKED = "booked"
    HOLD = "hold"
    SENT = "sent"

    def __str__(self) -> str:
        return str(self.value)

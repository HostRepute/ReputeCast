from enum import StrEnum

class SponsorBlockBookingStatus(StrEnum):
    AWAITING_PAYMENT = "awaiting_payment"
    BOOKED = "booked"
    HOLD = "hold"
    SENT = "sent"
    VALUE_4 = ""

    def __str__(self) -> str:
        return str(self.value)

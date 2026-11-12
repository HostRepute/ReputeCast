from enum import StrEnum

class SponsorBlockBookingStatus(StrEnum):
    BOOKED = "booked"
    HOLD = "hold"
    SENT = "sent"
    VALUE_3 = ""

    def __str__(self) -> str:
        return str(self.value)

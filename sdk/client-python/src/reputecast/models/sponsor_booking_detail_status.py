from enum import StrEnum

class SponsorBookingDetailStatus(StrEnum):
    BOOKED = "booked"
    CANCELED = "canceled"
    HOLD = "hold"
    PROPOSED = "proposed"
    SENT = "sent"

    def __str__(self) -> str:
        return str(self.value)

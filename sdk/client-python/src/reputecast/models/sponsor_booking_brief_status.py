from enum import StrEnum

class SponsorBookingBriefStatus(StrEnum):
    BOOKED = "booked"
    HOLD = "hold"
    SENT = "sent"

    def __str__(self) -> str:
        return str(self.value)

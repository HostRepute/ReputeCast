from enum import StrEnum

class SponsorBookingDetailCreativeStatus(StrEnum):
    READY = "ready"
    WAITING = "waiting"

    def __str__(self) -> str:
        return str(self.value)

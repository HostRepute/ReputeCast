from enum import StrEnum

class SponsorBookingBriefCreativeStatus(StrEnum):
    READY = "ready"
    WAITING = "waiting"

    def __str__(self) -> str:
        return str(self.value)

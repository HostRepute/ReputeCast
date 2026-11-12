from enum import StrEnum

class SponsorBlockState(StrEnum):
    NONE = "none"
    READY = "ready"
    WAITING = "waiting"

    def __str__(self) -> str:
        return str(self.value)

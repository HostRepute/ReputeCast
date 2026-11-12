from enum import StrEnum

class DigestDetailState(StrEnum):
    OFF = "off"
    ON = "on"
    PAUSED = "paused"

    def __str__(self) -> str:
        return str(self.value)

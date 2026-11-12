from enum import StrEnum

class DigestListItemState(StrEnum):
    OFF = "off"
    ON = "on"
    PAUSED = "paused"

    def __str__(self) -> str:
        return str(self.value)

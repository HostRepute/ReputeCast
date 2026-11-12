from enum import StrEnum

class ConnectStreamChoiceInputRole(StrEnum):
    FALLBACK = "fallback"
    PRIMARY = "primary"
    SPLIT = "split"

    def __str__(self) -> str:
        return str(self.value)

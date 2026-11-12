from enum import StrEnum

class ConnectStreamViewRole(StrEnum):
    FALLBACK = "fallback"
    PRIMARY = "primary"
    SPLIT = "split"

    def __str__(self) -> str:
        return str(self.value)

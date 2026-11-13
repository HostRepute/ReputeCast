from enum import StrEnum

class CountdownSettingsSize(StrEnum):
    L = "l"
    M = "m"
    S = "s"

    def __str__(self) -> str:
        return str(self.value)

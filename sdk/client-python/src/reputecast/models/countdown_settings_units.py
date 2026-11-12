from enum import StrEnum

class CountdownSettingsUnits(StrEnum):
    DHM = "dhm"
    DHMS = "dhms"
    HMS = "hms"

    def __str__(self) -> str:
        return str(self.value)

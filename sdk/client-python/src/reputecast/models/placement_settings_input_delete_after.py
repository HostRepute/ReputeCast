from enum import StrEnum

class PlacementSettingsInputDeleteAfter(StrEnum):
    FINISHED = "finished"
    NEVER = "never"
    VALUE_0 = "24h"

    def __str__(self) -> str:
        return str(self.value)

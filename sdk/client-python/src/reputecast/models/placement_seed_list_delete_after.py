from enum import StrEnum

class PlacementSeedListDeleteAfter(StrEnum):
    FINISHED = "finished"
    NEVER = "never"
    VALUE_0 = "24h"

    def __str__(self) -> str:
        return str(self.value)

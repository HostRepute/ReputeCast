from enum import StrEnum

class PlacementTestRowThrough(StrEnum):
    COMPARE = "compare"
    PROVIDER = "provider"
    ROUTE = "route"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class PlacementProviderOptionStatus(StrEnum):
    DEGRADED = "degraded"
    DOWN = "down"
    HEALTHY = "healthy"
    PAUSED = "paused"
    SANDBOX = "sandbox"
    WARMING_UP = "warming_up"

    def __str__(self) -> str:
        return str(self.value)

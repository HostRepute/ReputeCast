from enum import StrEnum

class ProviderOutageStreamState(StrEnum):
    FAILOVER_ACTIVE = "failover_active"
    FALLBACK_UNAVAILABLE = "fallback_unavailable"
    HELD = "held"
    WAITING = "waiting"

    def __str__(self) -> str:
        return str(self.value)

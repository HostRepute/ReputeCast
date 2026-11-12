from enum import StrEnum

class ReadinessRoutePolicy(StrEnum):
    AUTOMATIC = "automatic"
    PAUSE_AND_ASK = "pause_and_ask"
    WARM_ONLY = "warm_only"

    def __str__(self) -> str:
        return str(self.value)

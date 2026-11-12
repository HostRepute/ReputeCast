from enum import StrEnum

class DeliverabilityCheckRunTrigger(StrEnum):
    CONNECT = "connect"
    FALLBACK = "fallback"
    MANUAL = "manual"
    SCHEDULE = "schedule"

    def __str__(self) -> str:
        return str(self.value)

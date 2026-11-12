from enum import StrEnum

class DeliverabilityChecksInterval(StrEnum):
    DAILY = "daily"
    HOURLY = "hourly"

    def __str__(self) -> str:
        return str(self.value)

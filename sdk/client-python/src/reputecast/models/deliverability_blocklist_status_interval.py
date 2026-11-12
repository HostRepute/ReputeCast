from enum import StrEnum

class DeliverabilityBlocklistStatusInterval(StrEnum):
    DAILY = "daily"
    HOURLY = "hourly"

    def __str__(self) -> str:
        return str(self.value)

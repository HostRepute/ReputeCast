from enum import StrEnum

class ReadinessThresholdsScope(StrEnum):
    CAMPAIGN = "campaign"
    ROUTE = "route"

    def __str__(self) -> str:
        return str(self.value)

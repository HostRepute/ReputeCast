from enum import StrEnum

class LiveSnapshotThresholdScope(StrEnum):
    CAMPAIGN = "campaign"
    ROUTE = "route"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class RevenueUpgradeRowKind(StrEnum):
    AUTOMATION = "automation"
    CAMPAIGN = "campaign"
    NONE = "none"

    def __str__(self) -> str:
        return str(self.value)

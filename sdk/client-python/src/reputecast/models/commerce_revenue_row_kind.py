from enum import StrEnum

class CommerceRevenueRowKind(StrEnum):
    AUTOMATION = "automation"
    CAMPAIGN = "campaign"
    OTHERS = "others"

    def __str__(self) -> str:
        return str(self.value)

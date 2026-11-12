from enum import StrEnum

class CouponPoolUseKind(StrEnum):
    AUTOMATION = "automation"
    CAMPAIGN = "campaign"
    TEMPLATE = "template"

    def __str__(self) -> str:
        return str(self.value)

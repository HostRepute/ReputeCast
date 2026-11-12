from enum import StrEnum

class ReferralMilestoneViewType(StrEnum):
    CODES = "codes"
    COUPON = "coupon"
    DOWNLOAD = "download"
    MANUAL = "manual"
    WEBHOOK = "webhook"

    def __str__(self) -> str:
        return str(self.value)

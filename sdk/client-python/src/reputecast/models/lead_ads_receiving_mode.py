from enum import StrEnum

class LeadAdsReceivingMode(StrEnum):
    POLLING = "polling"
    WEBHOOK = "webhook"

    def __str__(self) -> str:
        return str(self.value)

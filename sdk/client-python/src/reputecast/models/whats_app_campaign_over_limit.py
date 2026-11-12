from enum import StrEnum

class WhatsAppCampaignOverLimit(StrEnum):
    NEXT_DAY = "next_day"
    SKIP = "skip"

    def __str__(self) -> str:
        return str(self.value)

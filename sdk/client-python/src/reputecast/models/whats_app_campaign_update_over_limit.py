from enum import StrEnum

class WhatsAppCampaignUpdateOverLimit(StrEnum):
    NEXT_DAY = "next_day"
    SKIP = "skip"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)

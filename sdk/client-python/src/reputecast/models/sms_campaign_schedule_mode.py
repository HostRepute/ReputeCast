from enum import StrEnum

class SmsCampaignScheduleMode(StrEnum):
    AT = "at"
    LOCAL = "local"
    NOW = "now"

    def __str__(self) -> str:
        return str(self.value)

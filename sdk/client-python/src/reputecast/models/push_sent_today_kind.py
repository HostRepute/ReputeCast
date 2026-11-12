from enum import StrEnum

class PushSentTodayKind(StrEnum):
    AUTOMATION = "automation"
    CAMPAIGN = "campaign"

    def __str__(self) -> str:
        return str(self.value)

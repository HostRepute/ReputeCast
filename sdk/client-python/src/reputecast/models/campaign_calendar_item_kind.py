from enum import StrEnum

class CampaignCalendarItemKind(StrEnum):
    DIGEST_ISSUE = "digest_issue"
    SCHEDULED = "scheduled"
    SENT = "sent"

    def __str__(self) -> str:
        return str(self.value)

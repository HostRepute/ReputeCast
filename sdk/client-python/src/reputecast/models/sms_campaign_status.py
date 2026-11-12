from enum import StrEnum

class SmsCampaignStatus(StrEnum):
    APPROVAL = "approval"
    APPROVED = "approved"
    CANCELLED = "cancelled"
    DRAFT = "draft"
    FAILED = "failed"
    PAUSED = "paused"
    SCHEDULED = "scheduled"
    SENDING = "sending"
    SENT = "sent"

    def __str__(self) -> str:
        return str(self.value)

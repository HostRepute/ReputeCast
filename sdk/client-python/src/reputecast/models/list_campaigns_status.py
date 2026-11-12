from enum import StrEnum

class ListCampaignsStatus(StrEnum):
    ALL = "all"
    ARCHIVED = "archived"
    DELETED = "deleted"
    DRAFTS = "drafts"
    SCHEDULED = "scheduled"
    SENDING = "sending"
    SENT = "sent"

    def __str__(self) -> str:
        return str(self.value)

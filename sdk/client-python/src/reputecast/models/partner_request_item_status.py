from enum import StrEnum

class PartnerRequestItemStatus(StrEnum):
    ACCEPTED = "accepted"
    FAILED = "failed"
    LIMIT = "limit"
    PAUSED = "paused"
    QUEUED = "queued"
    REJECTED = "rejected"

    def __str__(self) -> str:
        return str(self.value)

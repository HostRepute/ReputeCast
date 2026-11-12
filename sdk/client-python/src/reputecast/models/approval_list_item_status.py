from enum import StrEnum

class ApprovalListItemStatus(StrEnum):
    APPROVED = "approved"
    BYPASSED = "bypassed"
    CANCELLED = "cancelled"
    CHANGES_REQUESTED = "changes_requested"
    INVALIDATED = "invalidated"
    PENDING = "pending"
    REJECTED = "rejected"
    SENT = "sent"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class ListApprovalsStatus(StrEnum):
    ALL = "all"
    APPROVED = "approved"
    CHANGES_REQUESTED = "changes_requested"
    INVALIDATED = "invalidated"
    PENDING = "pending"
    REJECTED = "rejected"

    def __str__(self) -> str:
        return str(self.value)

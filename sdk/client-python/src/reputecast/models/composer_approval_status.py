from enum import StrEnum

class ComposerApprovalStatus(StrEnum):
    APPROVED = "approved"
    PENDING = "pending"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class WhatsAppTemplateStatus(StrEnum):
    APPROVED = "approved"
    DISABLED = "disabled"
    IN_APPEAL = "in_appeal"
    OTHER = "other"
    PAUSED = "paused"
    PENDING = "pending"
    REJECTED = "rejected"

    def __str__(self) -> str:
        return str(self.value)

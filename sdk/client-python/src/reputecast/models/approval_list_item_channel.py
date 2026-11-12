from enum import StrEnum

class ApprovalListItemChannel(StrEnum):
    EMAIL = "email"
    PUSH = "push"
    SMS = "sms"
    WHATSAPP = "whatsapp"

    def __str__(self) -> str:
        return str(self.value)

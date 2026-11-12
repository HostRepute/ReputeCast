from enum import StrEnum

class WhatsAppCheckKey(StrEnum):
    ACCOUNT = "account"
    APPROVAL = "approval"
    CONSENT = "consent"
    COST = "cost"
    LIMIT = "limit"
    QUALITY = "quality"
    TEMPLATE = "template"

    def __str__(self) -> str:
        return str(self.value)

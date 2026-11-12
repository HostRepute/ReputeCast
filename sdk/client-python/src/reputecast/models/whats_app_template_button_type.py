from enum import StrEnum

class WhatsAppTemplateButtonType(StrEnum):
    OTHER = "other"
    PHONE_NUMBER = "phone_number"
    QUICK_REPLY = "quick_reply"
    URL = "url"

    def __str__(self) -> str:
        return str(self.value)

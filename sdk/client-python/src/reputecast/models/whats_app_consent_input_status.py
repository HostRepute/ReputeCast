from enum import StrEnum

class WhatsAppConsentInputStatus(StrEnum):
    NOT_SUBSCRIBED = "not_subscribed"
    SUBSCRIBED = "subscribed"

    def __str__(self) -> str:
        return str(self.value)

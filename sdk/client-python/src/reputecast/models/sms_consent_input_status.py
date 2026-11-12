from enum import StrEnum

class SmsConsentInputStatus(StrEnum):
    NOT_SUBSCRIBED = "not_subscribed"
    SUBSCRIBED = "subscribed"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class SmsContactListSms(StrEnum):
    NOT_SUBSCRIBED = "not_subscribed"
    OPTED_OUT = "opted_out"
    PENDING = "pending"
    SUBSCRIBED = "subscribed"

    def __str__(self) -> str:
        return str(self.value)

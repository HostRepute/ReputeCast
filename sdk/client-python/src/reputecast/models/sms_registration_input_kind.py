from enum import StrEnum

class SmsRegistrationInputKind(StrEnum):
    BRAND = "brand"
    CAMPAIGN = "campaign"
    SENDER_ID = "sender_id"
    TOLL_FREE = "toll_free"

    def __str__(self) -> str:
        return str(self.value)

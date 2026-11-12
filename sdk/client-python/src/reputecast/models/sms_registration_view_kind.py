from enum import StrEnum

class SmsRegistrationViewKind(StrEnum):
    BRAND = "brand"
    CAMPAIGN = "campaign"
    SENDER_ID = "sender_id"
    TOLL_FREE = "toll_free"
    VALUE_4 = ""

    def __str__(self) -> str:
        return str(self.value)

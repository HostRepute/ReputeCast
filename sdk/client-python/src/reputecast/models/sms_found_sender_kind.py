from enum import StrEnum

class SmsFoundSenderKind(StrEnum):
    LOCAL = "local"
    MOBILE = "mobile"
    SENDER_ID = "sender_id"
    TOLL_FREE = "toll_free"
    VALUE_1 = "10dlc"

    def __str__(self) -> str:
        return str(self.value)

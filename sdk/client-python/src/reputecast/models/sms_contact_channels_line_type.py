from enum import StrEnum

class SmsContactChannelsLineType(StrEnum):
    LANDLINE = "landline"
    MOBILE = "mobile"
    TOLL_FREE = "toll_free"
    UNKNOWN = "unknown"
    VALUE_5 = ""
    VOIP = "voip"

    def __str__(self) -> str:
        return str(self.value)

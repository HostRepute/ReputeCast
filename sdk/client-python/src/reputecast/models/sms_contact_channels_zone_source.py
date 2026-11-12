from enum import StrEnum

class SmsContactChannelsZoneSource(StrEnum):
    AREA_CODE = "area_code"
    CONTACT = "contact"
    COUNTRY = "country"
    UNKNOWN = "unknown"

    def __str__(self) -> str:
        return str(self.value)

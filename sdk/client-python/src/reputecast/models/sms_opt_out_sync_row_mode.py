from enum import StrEnum

class SmsOptOutSyncRowMode(StrEnum):
    OFF = "off"
    READ_ONLY = "read_only"
    TWO_WAY = "two_way"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class CouponSourceReason(StrEnum):
    NOT_CONNECTED = "not_connected"
    NOT_GRANTED = "not_granted"
    READ_ONLY = "read_only"
    UNSUPPORTED = "unsupported"
    VALUE_4 = ""

    def __str__(self) -> str:
        return str(self.value)

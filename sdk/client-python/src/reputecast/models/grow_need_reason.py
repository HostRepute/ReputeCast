from enum import StrEnum

class GrowNeedReason(StrEnum):
    ENDED = "ended"
    SIGNATURE_REJECTED = "signature_rejected"
    UNKNOWN_LINK = "unknown_link"
    VALUE_3 = ""

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class DeliverabilityDMARCSummaryVia(StrEnum):
    HOSTREPUTE = "hostrepute"
    NONE = "none"
    UPLOAD = "upload"

    def __str__(self) -> str:
        return str(self.value)

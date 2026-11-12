from enum import StrEnum

class DeliverabilityDMARCSummaryVia(StrEnum):
    HOSTREPUTE = "hostrepute"
    INBOUND = "inbound"
    NONE = "none"
    UPLOAD = "upload"

    def __str__(self) -> str:
        return str(self.value)

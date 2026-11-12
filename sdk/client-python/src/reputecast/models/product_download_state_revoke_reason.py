from enum import StrEnum

class ProductDownloadStateRevokeReason(StrEnum):
    DISPUTE = "dispute"
    MANUAL = "manual"
    REFUND = "refund"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)

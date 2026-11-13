from enum import StrEnum

class CouponBatchKind(StrEnum):
    GENERATE = "generate"
    UPLOAD = "upload"

    def __str__(self) -> str:
        return str(self.value)

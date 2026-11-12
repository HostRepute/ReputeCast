from enum import StrEnum

class CouponBatchReason(StrEnum):
    POOL = "pool"
    REFILL = "refill"
    REQUEST = "request"

    def __str__(self) -> str:
        return str(self.value)

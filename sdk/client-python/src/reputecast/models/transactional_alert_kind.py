from enum import StrEnum

class TransactionalAlertKind(StrEnum):
    BOUNCE_RATE = "bounce_rate"
    FAILED = "failed"
    FAILOVER = "failover"
    MEDIAN_DELIVERY = "median_delivery"

    def __str__(self) -> str:
        return str(self.value)

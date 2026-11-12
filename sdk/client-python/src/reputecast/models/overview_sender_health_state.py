from enum import StrEnum

class OverviewSenderHealthState(StrEnum):
    PENDING = "pending"
    READY = "ready"

    def __str__(self) -> str:
        return str(self.value)

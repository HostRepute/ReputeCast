from enum import StrEnum

class DeliverabilityBlocklistSourceDelisting(StrEnum):
    DELISTED = "delisted"
    DENIED = "denied"
    PENDING = "pending"
    REQUESTED = "requested"

    def __str__(self) -> str:
        return str(self.value)

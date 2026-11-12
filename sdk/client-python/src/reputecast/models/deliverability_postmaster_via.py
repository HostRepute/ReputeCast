from enum import StrEnum

class DeliverabilityPostmasterVia(StrEnum):
    DIRECT = "direct"
    HOSTREPUTE = "hostrepute"

    def __str__(self) -> str:
        return str(self.value)

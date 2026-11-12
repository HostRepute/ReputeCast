from enum import StrEnum

class DeliverabilityCheckRunVia(StrEnum):
    BUILTIN = "builtin"
    HOSTREPUTE = "hostrepute"

    def __str__(self) -> str:
        return str(self.value)

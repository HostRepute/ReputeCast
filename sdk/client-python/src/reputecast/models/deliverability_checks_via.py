from enum import StrEnum

class DeliverabilityChecksVia(StrEnum):
    BUILTIN = "builtin"
    HOSTREPUTE = "hostrepute"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class DeliverabilityReputationVia(StrEnum):
    BUILTIN = "builtin"
    HOSTREPUTE = "hostrepute"
    NONE = "none"

    def __str__(self) -> str:
        return str(self.value)

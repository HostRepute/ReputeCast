from enum import StrEnum

class DeliverabilityIncidentSource(StrEnum):
    BUILTIN = "builtin"
    HOSTREPUTE = "hostrepute"
    REPUTECAST = "reputecast"

    def __str__(self) -> str:
        return str(self.value)

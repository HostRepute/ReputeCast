from enum import StrEnum

class DeliverabilityAssetBlocklistsVia(StrEnum):
    BUILTIN = "builtin"
    HOSTREPUTE = "hostrepute"
    NONE = "none"

    def __str__(self) -> str:
        return str(self.value)

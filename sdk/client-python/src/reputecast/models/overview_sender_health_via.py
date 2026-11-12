from enum import StrEnum

class OverviewSenderHealthVia(StrEnum):
    BUILTIN = "builtin"
    HOSTREPUTE = "hostrepute"
    NONE = "none"

    def __str__(self) -> str:
        return str(self.value)

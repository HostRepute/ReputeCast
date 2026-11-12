from enum import StrEnum

class DeliverabilitySNDSVia(StrEnum):
    DIRECT = "direct"
    HOSTREPUTE = "hostrepute"

    def __str__(self) -> str:
        return str(self.value)

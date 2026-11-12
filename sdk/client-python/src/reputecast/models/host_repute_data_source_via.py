from enum import StrEnum

class HostReputeDataSourceVia(StrEnum):
    DIRECT = "direct"
    HOSTREPUTE = "hostrepute"
    NONE = "none"

    def __str__(self) -> str:
        return str(self.value)

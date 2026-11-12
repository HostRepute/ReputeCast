from enum import StrEnum

class ProviderCardIpMode(StrEnum):
    DEDICATED = "dedicated"
    SHARED = "shared"

    def __str__(self) -> str:
        return str(self.value)

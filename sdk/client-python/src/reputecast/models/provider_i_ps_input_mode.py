from enum import StrEnum

class ProviderIPsInputMode(StrEnum):
    DEDICATED = "dedicated"
    SHARED = "shared"

    def __str__(self) -> str:
        return str(self.value)

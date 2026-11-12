from enum import StrEnum

class ProviderIPSourceFrom(StrEnum):
    MANUAL = "manual"
    NONE = "none"
    PROVIDER = "provider"

    def __str__(self) -> str:
        return str(self.value)

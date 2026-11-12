from enum import StrEnum

class ProviderTrackingCheckMode(StrEnum):
    PROVIDER = "provider"
    REPUTECAST = "reputecast"

    def __str__(self) -> str:
        return str(self.value)

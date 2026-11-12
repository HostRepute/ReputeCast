from enum import StrEnum

class ProviderTrackingActionInputAction(StrEnum):
    DISABLE = "disable"
    KEEP_PROVIDER = "keep_provider"
    RECHECK = "recheck"
    USE_REPUTECAST = "use_reputecast"

    def __str__(self) -> str:
        return str(self.value)

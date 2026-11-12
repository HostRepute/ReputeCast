from enum import StrEnum

class ProviderDownStateReason(StrEnum):
    AUTH = "auth"
    OUTAGE = "outage"

    def __str__(self) -> str:
        return str(self.value)

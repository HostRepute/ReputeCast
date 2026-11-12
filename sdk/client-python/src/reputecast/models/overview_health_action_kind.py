from enum import StrEnum

class OverviewHealthActionKind(StrEnum):
    DELISTING = "delisting"
    DELIVERABILITY = "deliverability"
    INTEGRATIONS = "integrations"
    PROVIDERS = "providers"
    WARMUP = "warmup"

    def __str__(self) -> str:
        return str(self.value)

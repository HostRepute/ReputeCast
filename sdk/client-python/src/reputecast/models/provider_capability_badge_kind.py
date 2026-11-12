from enum import StrEnum

class ProviderCapabilityBadgeKind(StrEnum):
    INFO = "info"
    NO = "no"
    PARTIAL = "partial"
    YES = "yes"

    def __str__(self) -> str:
        return str(self.value)

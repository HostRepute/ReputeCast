from enum import StrEnum

class ReportProviderColumnTrackedBy(StrEnum):
    PROVIDER = "provider"
    REPUTECAST = "reputecast"

    def __str__(self) -> str:
        return str(self.value)

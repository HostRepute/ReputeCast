from enum import StrEnum

class LeadAdsActivityItemSource(StrEnum):
    IMPORT = "import"
    POLL = "poll"
    WEBHOOK = "webhook"

    def __str__(self) -> str:
        return str(self.value)

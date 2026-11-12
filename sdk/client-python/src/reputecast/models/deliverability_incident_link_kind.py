from enum import StrEnum

class DeliverabilityIncidentLinkKind(StrEnum):
    CAMPAIGN = "campaign"
    CAMPAIGN_REPORT = "campaign_report"
    DELIVERABILITY = "deliverability"
    IMPORT = "import"
    PROVIDER = "provider"
    PROVIDER_EVENTS = "provider_events"
    WARMUP = "warmup"

    def __str__(self) -> str:
        return str(self.value)

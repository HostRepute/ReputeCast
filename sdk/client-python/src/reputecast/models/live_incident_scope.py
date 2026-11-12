from enum import StrEnum

class LiveIncidentScope(StrEnum):
    CAMPAIGN = "campaign"
    MAILBOX_PROVIDER = "mailbox_provider"
    ROUTE = "route"

    def __str__(self) -> str:
        return str(self.value)

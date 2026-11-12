from enum import StrEnum

class IntegrationStoreFactsState(StrEnum):
    CONNECTED = "connected"
    CONNECTING = "connecting"
    IMPORTING = "importing"
    NO_PLUGIN = "no_plugin"
    RATE_LIMITED = "rate_limited"
    REVOKED = "revoked"
    WEBHOOK_MISSING = "webhook_missing"

    def __str__(self) -> str:
        return str(self.value)

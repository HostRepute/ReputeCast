from enum import StrEnum

class CommerceSyncStateState(StrEnum):
    CONNECTED = "connected"
    CONNECTING = "connecting"
    DISCONNECTED = "disconnected"
    IMPORTING = "importing"
    NO_PLUGIN = "no_plugin"
    RATE_LIMITED = "rate_limited"
    WEBHOOK_MISSING = "webhook_missing"

    def __str__(self) -> str:
        return str(self.value)

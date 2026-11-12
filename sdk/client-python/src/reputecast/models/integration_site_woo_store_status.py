from enum import StrEnum

class IntegrationSiteWooStoreStatus(StrEnum):
    CONNECTED = "connected"
    CONNECTING = "connecting"
    DISCONNECTED = "disconnected"
    IMPORTING = "importing"
    VALUE_4 = ""

    def __str__(self) -> str:
        return str(self.value)

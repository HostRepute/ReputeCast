from enum import StrEnum

class CommerceStoreSummaryStatus(StrEnum):
    CONNECTED = "connected"
    CONNECTING = "connecting"
    DISCONNECTED = "disconnected"
    IMPORTING = "importing"

    def __str__(self) -> str:
        return str(self.value)

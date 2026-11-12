from enum import StrEnum

class ConnectWebhookTestStatus(StrEnum):
    ARRIVED = "arrived"
    MISSING = "missing"
    REFUSED = "refused"
    WAITING = "waiting"

    def __str__(self) -> str:
        return str(self.value)

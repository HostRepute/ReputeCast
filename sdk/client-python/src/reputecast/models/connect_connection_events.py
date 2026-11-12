from enum import StrEnum

class ConnectConnectionEvents(StrEnum):
    QUEUE = "queue"
    WEBHOOK = "webhook"

    def __str__(self) -> str:
        return str(self.value)

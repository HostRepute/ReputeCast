from enum import StrEnum

class ConnectSESInputEvents(StrEnum):
    QUEUE = "queue"
    WEBHOOK = "webhook"

    def __str__(self) -> str:
        return str(self.value)

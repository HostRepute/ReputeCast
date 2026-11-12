from enum import StrEnum

class ConnectEventsInputEvents(StrEnum):
    QUEUE = "queue"
    WEBHOOK = "webhook"

    def __str__(self) -> str:
        return str(self.value)

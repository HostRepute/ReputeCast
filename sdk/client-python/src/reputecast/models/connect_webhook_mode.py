from enum import StrEnum

class ConnectWebhookMode(StrEnum):
    MAILBOX = "mailbox"
    NONE = "none"
    POLLING = "polling"
    WEBHOOK = "webhook"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class ProviderLogEventChannel(StrEnum):
    BACKFILL = "backfill"
    BOUNCE_MAILBOX = "bounce_mailbox"
    INTERNAL = "internal"
    QUEUE = "queue"
    TEST = "test"
    WEBHOOK = "webhook"

    def __str__(self) -> str:
        return str(self.value)

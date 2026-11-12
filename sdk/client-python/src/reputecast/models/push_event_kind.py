from enum import StrEnum

class PushEventKind(StrEnum):
    CLICKED = "clicked"
    CLOSED = "closed"
    DELIVERED = "delivered"
    FAILED = "failed"
    GONE = "gone"
    HELP = "help"
    OPTED_IN = "opted_in"
    OPTED_OUT = "opted_out"
    QUEUED = "queued"
    READ = "read"
    REPLIED = "replied"
    SENT = "sent"
    SHOWN = "shown"
    UNDELIVERED = "undelivered"

    def __str__(self) -> str:
        return str(self.value)

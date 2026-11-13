from enum import StrEnum

class InboundRouteViewReplyHandling(StrEnum):
    FORWARD = "forward"
    INBOX = "inbox"

    def __str__(self) -> str:
        return str(self.value)

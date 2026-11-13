from enum import StrEnum

class InboundRouteViewKind(StrEnum):
    MAILBOX = "mailbox"
    PROVIDER = "provider"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class InboundMailboxViewSecurity(StrEnum):
    STARTTLS = "starttls"
    TLS = "tls"

    def __str__(self) -> str:
        return str(self.value)

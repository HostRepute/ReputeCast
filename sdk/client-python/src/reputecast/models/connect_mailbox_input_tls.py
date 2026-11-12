from enum import StrEnum

class ConnectMailboxInputTls(StrEnum):
    NONE = "none"
    STARTTLS = "starttls"
    TLS = "tls"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class TransactionalRelayCheckKey(StrEnum):
    CERTIFICATE = "certificate"
    DNS = "dns"
    STARTTLS = "starttls"
    TLS = "tls"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class TransactionalRelayState(StrEnum):
    CERTIFICATE = "certificate"
    OFF = "off"
    OK = "ok"
    UNCHECKED = "unchecked"
    UNREACHABLE = "unreachable"

    def __str__(self) -> str:
        return str(self.value)

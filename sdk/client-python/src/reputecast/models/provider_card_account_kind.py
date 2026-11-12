from enum import StrEnum

class ProviderCardAccountKind(StrEnum):
    ACCOUNT = "account"
    SERVER = "server"

    def __str__(self) -> str:
        return str(self.value)

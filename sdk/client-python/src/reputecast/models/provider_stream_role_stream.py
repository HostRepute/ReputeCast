from enum import StrEnum

class ProviderStreamRoleStream(StrEnum):
    AUTOMATION = "automation"
    BROADCAST = "broadcast"
    SYSTEM = "system"
    TRANSACTIONAL = "transactional"

    def __str__(self) -> str:
        return str(self.value)

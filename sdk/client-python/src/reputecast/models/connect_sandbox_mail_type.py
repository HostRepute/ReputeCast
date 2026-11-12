from enum import StrEnum

class ConnectSandboxMailType(StrEnum):
    MARKETING = "marketing"
    TRANSACTIONAL = "transactional"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class MCPAgentRowStatus(StrEnum):
    ACTIVE = "active"
    EXPIRED = "expired"

    def __str__(self) -> str:
        return str(self.value)

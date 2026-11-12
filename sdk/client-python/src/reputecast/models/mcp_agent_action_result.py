from enum import StrEnum

class MCPAgentActionResult(StrEnum):
    ALLOWED = "allowed"
    DENIED = "denied"
    ERROR = "error"

    def __str__(self) -> str:
        return str(self.value)

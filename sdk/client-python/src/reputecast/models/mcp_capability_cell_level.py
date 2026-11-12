from enum import StrEnum

class MCPCapabilityCellLevel(StrEnum):
    ALLOW = "allow"
    APPROVE = "approve"
    DENY = "deny"
    PARTIAL = "partial"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class ExportAuditEventsActorType(StrEnum):
    API_KEY = "api_key"
    MCP_AGENT = "mcp_agent"
    SYSTEM = "system"
    USER = "user"

    def __str__(self) -> str:
        return str(self.value)

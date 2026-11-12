from enum import StrEnum

class WorkspaceMeActorType(StrEnum):
    API_KEY = "api_key"
    MCP_AGENT = "mcp_agent"
    USER = "user"

    def __str__(self) -> str:
        return str(self.value)

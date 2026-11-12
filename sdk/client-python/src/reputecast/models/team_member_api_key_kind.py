from enum import StrEnum

class TeamMemberAPIKeyKind(StrEnum):
    API_KEY = "api_key"
    MCP_AGENT = "mcp_agent"

    def __str__(self) -> str:
        return str(self.value)

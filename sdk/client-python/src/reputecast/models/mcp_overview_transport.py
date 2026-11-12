from enum import StrEnum

class MCPOverviewTransport(StrEnum):
    STREAMABLE_HTTP = "streamable_http"

    def __str__(self) -> str:
        return str(self.value)

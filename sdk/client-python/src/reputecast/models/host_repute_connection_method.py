from enum import StrEnum

class HostReputeConnectionMethod(StrEnum):
    API_KEY = "api_key"
    OAUTH = "oauth"

    def __str__(self) -> str:
        return str(self.value)

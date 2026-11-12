from enum import StrEnum

class ProviderWebhookCheckKey(StrEnum):
    API = "api"
    ENDPOINT = "endpoint"
    PROVIDER_LOG = "provider_log"
    REGISTERED = "registered"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class ConnectCatalogEntryKind(StrEnum):
    API = "api"
    COMMUNITY = "community"
    SELF_HOSTED = "self_hosted"
    SMTP = "smtp"

    def __str__(self) -> str:
        return str(self.value)

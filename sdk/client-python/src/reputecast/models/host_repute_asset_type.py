from enum import StrEnum

class HostReputeAssetType(StrEnum):
    DOMAIN = "domain"
    HOSTNAME = "hostname"
    IP = "ip"

    def __str__(self) -> str:
        return str(self.value)

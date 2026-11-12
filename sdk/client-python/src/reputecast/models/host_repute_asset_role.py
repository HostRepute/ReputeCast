from enum import StrEnum

class HostReputeAssetRole(StrEnum):
    DEDICATED_IP = "dedicated_ip"
    ORGANIZATION_DOMAIN = "organization_domain"
    OTHER = "other"
    SENDING_DOMAIN = "sending_domain"
    SERVER_HOSTNAME = "server_hostname"

    def __str__(self) -> str:
        return str(self.value)

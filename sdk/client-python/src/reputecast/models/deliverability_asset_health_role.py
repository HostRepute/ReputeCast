from enum import StrEnum

class DeliverabilityAssetHealthRole(StrEnum):
    DEDICATED_IP = "dedicated_ip"
    SENDING_DOMAIN = "sending_domain"

    def __str__(self) -> str:
        return str(self.value)

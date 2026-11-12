from enum import StrEnum

class DeliverabilityAssetHealthType(StrEnum):
    DOMAIN = "domain"
    IP = "ip"

    def __str__(self) -> str:
        return str(self.value)

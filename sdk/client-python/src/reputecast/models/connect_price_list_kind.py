from enum import StrEnum

class ConnectPriceListKind(StrEnum):
    CONTRACT = "contract"
    METERED = "metered"
    PLANS = "plans"
    RELAY = "relay"
    SELF_HOSTED = "self_hosted"

    def __str__(self) -> str:
        return str(self.value)

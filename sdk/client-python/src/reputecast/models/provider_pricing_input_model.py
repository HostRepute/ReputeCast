from enum import StrEnum

class ProviderPricingInputModel(StrEnum):
    METERED = "metered"
    PLAN = "plan"
    SELF_HOSTED = "self_hosted"

    def __str__(self) -> str:
        return str(self.value)

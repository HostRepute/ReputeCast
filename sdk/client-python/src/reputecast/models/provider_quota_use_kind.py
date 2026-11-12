from enum import StrEnum

class ProviderQuotaUseKind(StrEnum):
    MONTHLY_PLAN = "monthly_plan"
    NONE = "none"
    ROLLING_24H = "rolling_24h"
    WARMUP_DAILY = "warmup_daily"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class ReferralMilestoneHealthKind(StrEnum):
    CODES_LOW = "codes_low"
    CODES_OUT = "codes_out"
    FILE_MISSING = "file_missing"
    HEALTHY = "healthy"
    STRIPE_MISSING = "stripe_missing"
    WEBHOOK_FAILING = "webhook_failing"

    def __str__(self) -> str:
        return str(self.value)

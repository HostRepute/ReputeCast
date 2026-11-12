from enum import StrEnum

class GrowNeedKind(StrEnum):
    CODES_LOW = "codes_low"
    CODES_OUT = "codes_out"
    FILE_MISSING = "file_missing"
    HELD = "held"
    PARTNER_INVALID = "partner_invalid"
    PARTNER_UNREACHABLE = "partner_unreachable"
    STRIPE_MISSING = "stripe_missing"
    TO_FULFILL = "to_fulfill"
    UNCLAIMED = "unclaimed"
    WEBHOOK_FAILING = "webhook_failing"

    def __str__(self) -> str:
        return str(self.value)

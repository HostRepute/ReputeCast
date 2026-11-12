from enum import StrEnum

class CanaryGateKey(StrEnum):
    BLOCK_BOUNCE = "block_bounce"
    BOUNCE = "bounce"
    COMPLAINT = "complaint"
    DEFERRAL = "deferral"
    SEEDS = "seeds"

    def __str__(self) -> str:
        return str(self.value)

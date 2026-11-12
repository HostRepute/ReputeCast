from enum import StrEnum

class LiveWarmupHoldDecision(StrEnum):
    WAIT = "wait"

    def __str__(self) -> str:
        return str(self.value)

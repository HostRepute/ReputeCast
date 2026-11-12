from enum import StrEnum

class CanaryGateState(StrEnum):
    COLLECTING = "collecting"
    FAILING = "failing"
    PASSING = "passing"

    def __str__(self) -> str:
        return str(self.value)

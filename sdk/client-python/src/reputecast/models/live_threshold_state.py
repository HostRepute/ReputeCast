from enum import StrEnum

class LiveThresholdState(StrEnum):
    COLLECTING = "collecting"
    NA = "na"
    NEAR = "near"
    OK = "ok"
    TRIPPED = "tripped"

    def __str__(self) -> str:
        return str(self.value)

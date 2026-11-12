from enum import StrEnum

class RouteTestStepKind(StrEnum):
    CONDITION = "condition"
    HELD = "held"
    PRIMARY = "primary"
    SPLIT = "split"
    STREAM = "stream"
    SUPPRESSION = "suppression"

    def __str__(self) -> str:
        return str(self.value)

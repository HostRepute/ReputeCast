from enum import StrEnum

class ScoringFlowImpactDirection(StrEnum):
    FALLS = "falls"
    RISES = "rises"

    def __str__(self) -> str:
        return str(self.value)

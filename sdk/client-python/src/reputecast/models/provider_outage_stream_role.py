from enum import StrEnum

class ProviderOutageStreamRole(StrEnum):
    CONDITION = "condition"
    FALLBACK = "fallback"
    PRIMARY = "primary"
    SPLIT = "split"

    def __str__(self) -> str:
        return str(self.value)

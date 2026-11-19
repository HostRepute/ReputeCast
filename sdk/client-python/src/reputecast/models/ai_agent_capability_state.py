from enum import StrEnum

class AIAgentCapabilityState(StrEnum):
    SUPPORTED = "supported"
    UNSUPPORTED = "unsupported"
    UNTESTED = "untested"

    def __str__(self) -> str:
        return str(self.value)

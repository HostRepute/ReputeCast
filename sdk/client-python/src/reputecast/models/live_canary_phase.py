from enum import StrEnum

class LiveCanaryPhase(StrEnum):
    CONTINUED = "continued"
    HELD = "held"
    PASSED = "passed"
    SENDING = "sending"
    STOPPED = "stopped"
    WATCHING = "watching"

    def __str__(self) -> str:
        return str(self.value)

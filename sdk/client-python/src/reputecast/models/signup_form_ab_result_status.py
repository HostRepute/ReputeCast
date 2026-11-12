from enum import StrEnum

class SignupFormABResultStatus(StrEnum):
    ENDED = "ended"
    NONE = "none"
    RUNNING = "running"

    def __str__(self) -> str:
        return str(self.value)

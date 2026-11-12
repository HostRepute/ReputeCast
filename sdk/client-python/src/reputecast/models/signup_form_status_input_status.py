from enum import StrEnum

class SignupFormStatusInputStatus(StrEnum):
    LIVE = "live"
    PAUSED = "paused"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class EarnCardState(StrEnum):
    INVITE = "invite"
    LIVE = "live"
    NONE = "none"

    def __str__(self) -> str:
        return str(self.value)

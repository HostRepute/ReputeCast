from enum import StrEnum

class GrowBlockRenderInputKind(StrEnum):
    REFERRAL = "referral"
    REWARD = "reward"

    def __str__(self) -> str:
        return str(self.value)

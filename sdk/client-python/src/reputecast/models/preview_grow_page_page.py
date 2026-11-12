from enum import StrEnum

class PreviewGrowPagePage(StrEnum):
    HUB = "hub"
    INVITE = "invite"
    LEADERBOARD = "leaderboard"
    RECOMMENDS = "recommends"

    def __str__(self) -> str:
        return str(self.value)

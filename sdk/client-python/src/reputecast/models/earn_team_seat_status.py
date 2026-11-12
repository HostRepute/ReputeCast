from enum import StrEnum

class EarnTeamSeatStatus(StrEnum):
    ACTIVE = "active"
    BUYER = "buyer"
    INVITED = "invited"

    def __str__(self) -> str:
        return str(self.value)

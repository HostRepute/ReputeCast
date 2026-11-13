from enum import StrEnum

class EarnMemberTeamRole(StrEnum):
    BUYER = "buyer"
    SEAT = "seat"

    def __str__(self) -> str:
        return str(self.value)

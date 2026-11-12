from enum import StrEnum

class TeamMemberStatus(StrEnum):
    ACTIVE = "active"
    DEACTIVATED = "deactivated"

    def __str__(self) -> str:
        return str(self.value)

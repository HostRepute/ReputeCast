from enum import StrEnum

class RoleMemberRefStatus(StrEnum):
    ACTIVE = "active"
    DEACTIVATED = "deactivated"

    def __str__(self) -> str:
        return str(self.value)

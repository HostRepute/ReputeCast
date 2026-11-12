from enum import StrEnum

class WorkspaceAgencyStaffLevel(StrEnum):
    ACCOUNT_MANAGER = "account_manager"
    AGENCY_MANAGER = "agency_manager"

    def __str__(self) -> str:
        return str(self.value)

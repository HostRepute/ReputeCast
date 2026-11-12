from enum import StrEnum

class WorkspaceAgencyKind(StrEnum):
    AGENCY = "agency"
    CLIENT = "client"

    def __str__(self) -> str:
        return str(self.value)

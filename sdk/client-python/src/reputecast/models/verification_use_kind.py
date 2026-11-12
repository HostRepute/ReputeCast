from enum import StrEnum

class VerificationUseKind(StrEnum):
    CONTACT = "contact"
    IMPORT = "import"
    LIST = "list"
    REALTIME = "realtime"
    SEGMENT = "segment"
    SELECTION = "selection"
    TIER = "tier"

    def __str__(self) -> str:
        return str(self.value)

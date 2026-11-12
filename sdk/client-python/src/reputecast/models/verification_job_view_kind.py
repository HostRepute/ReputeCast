from enum import StrEnum

class VerificationJobViewKind(StrEnum):
    CONTACT = "contact"
    IMPORT = "import"
    LIST = "list"
    SEGMENT = "segment"
    SELECTION = "selection"
    TIER = "tier"

    def __str__(self) -> str:
        return str(self.value)

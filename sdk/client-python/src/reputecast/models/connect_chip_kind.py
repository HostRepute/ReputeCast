from enum import StrEnum

class ConnectChipKind(StrEnum):
    NO = "no"
    PARTIAL = "partial"
    YES = "yes"

    def __str__(self) -> str:
        return str(self.value)

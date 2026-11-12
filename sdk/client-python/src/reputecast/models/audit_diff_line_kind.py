from enum import StrEnum

class AuditDiffLineKind(StrEnum):
    ADD = "add"
    DEL = "del"
    SAME = "same"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class AuditRestoreKind(StrEnum):
    LIST = "list"

    def __str__(self) -> str:
        return str(self.value)

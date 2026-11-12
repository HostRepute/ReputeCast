from enum import StrEnum

class EarnMemberListAccessRule(StrEnum):
    END = "end"
    KEEP = "keep"

    def __str__(self) -> str:
        return str(self.value)

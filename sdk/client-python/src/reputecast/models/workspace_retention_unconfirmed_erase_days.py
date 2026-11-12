from enum import IntEnum

class WorkspaceRetentionUnconfirmedEraseDays(IntEnum):
    VALUE_0 = 0
    VALUE_14 = 14
    VALUE_30 = 30
    VALUE_60 = 60
    VALUE_90 = 90

    def __str__(self) -> str:
        return str(self.value)

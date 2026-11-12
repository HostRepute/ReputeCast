from enum import IntEnum

class WorkspaceRetentionEventDetailDays(IntEnum):
    VALUE_0 = 0
    VALUE_30 = 30
    VALUE_90 = 90
    VALUE_180 = 180
    VALUE_365 = 365
    VALUE_730 = 730

    def __str__(self) -> str:
        return str(self.value)

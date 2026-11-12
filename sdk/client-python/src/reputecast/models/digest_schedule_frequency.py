from enum import StrEnum

class DigestScheduleFrequency(StrEnum):
    DAILY = "daily"
    ITEMS = "items"
    MONTHLY = "monthly"
    WEEKLY = "weekly"

    def __str__(self) -> str:
        return str(self.value)

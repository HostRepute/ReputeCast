from enum import StrEnum

class ContactProfilePreferencesFrequency(StrEnum):
    ALL = "all"
    MONTHLY = "monthly"
    WEEKLY = "weekly"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class AutomationTriggerDirection(StrEnum):
    ANY = "any"
    DOWNGRADE = "downgrade"
    FALLS = "falls"
    RISES = "rises"
    UPGRADE = "upgrade"

    def __str__(self) -> str:
        return str(self.value)

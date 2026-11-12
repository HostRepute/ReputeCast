from enum import StrEnum

class IntegrationManagedHookStatus(StrEnum):
    DEAD_LETTERS = "dead_letters"
    FAILING = "failing"
    HEALTHY = "healthy"
    NEW = "new"

    def __str__(self) -> str:
        return str(self.value)

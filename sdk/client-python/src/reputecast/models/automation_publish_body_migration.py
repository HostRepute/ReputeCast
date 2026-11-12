from enum import StrEnum

class AutomationPublishBodyMigration(StrEnum):
    FINISH = "finish"
    MOVE = "move"

    def __str__(self) -> str:
        return str(self.value)

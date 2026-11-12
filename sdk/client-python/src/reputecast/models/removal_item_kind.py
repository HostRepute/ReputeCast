from enum import StrEnum

class RemovalItemKind(StrEnum):
    AUTOMATION_EMAIL = "automation_email"
    CAMPAIGN = "campaign"
    TRANSACTIONAL_TEMPLATE = "transactional_template"

    def __str__(self) -> str:
        return str(self.value)

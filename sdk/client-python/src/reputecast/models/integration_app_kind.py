from enum import StrEnum

class IntegrationAppKind(StrEnum):
    MAKE = "make"
    N8N = "n8n"
    ZAPIER = "zapier"

    def __str__(self) -> str:
        return str(self.value)

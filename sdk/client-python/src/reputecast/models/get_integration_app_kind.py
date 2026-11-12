from enum import StrEnum

class GetIntegrationAppKind(StrEnum):
    MAKE = "make"
    N8N = "n8n"
    WORDPRESS = "wordpress"
    ZAPIER = "zapier"

    def __str__(self) -> str:
        return str(self.value)

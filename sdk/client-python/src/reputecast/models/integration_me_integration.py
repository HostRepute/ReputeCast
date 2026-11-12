from enum import StrEnum

class IntegrationMeIntegration(StrEnum):
    MAKE = "make"
    N8N = "n8n"
    VALUE_4 = ""
    WORDPRESS = "wordpress"
    ZAPIER = "zapier"

    def __str__(self) -> str:
        return str(self.value)

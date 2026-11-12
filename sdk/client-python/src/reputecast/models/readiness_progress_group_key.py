from enum import StrEnum

class ReadinessProgressGroupKey(StrEnum):
    ACCESSIBILITY = "accessibility"
    AUDIENCE = "audience"
    AUTHENTICATION = "authentication"
    COMPLIANCE = "compliance"
    CONDITIONAL = "conditional"
    CONTENT = "content"
    LANGUAGES = "languages"
    LINKS = "links"
    REPUTATION = "reputation"
    ROUTING = "routing"
    RULES = "rules"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class SmsRulesInputKeywordsBy(StrEnum):
    PROVIDER = "provider"
    REPUTECAST = "reputecast"

    def __str__(self) -> str:
        return str(self.value)

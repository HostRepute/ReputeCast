from enum import StrEnum

class AIProviderEntryKind(StrEnum):
    ANTHROPIC = "anthropic"
    AZURE = "azure"
    COMPATIBLE = "compatible"
    GOOGLE = "google"
    OPENAI = "openai"

    def __str__(self) -> str:
        return str(self.value)

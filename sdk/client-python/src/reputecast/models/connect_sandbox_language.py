from enum import StrEnum

class ConnectSandboxLanguage(StrEnum):
    EN = "en"
    JA = "ja"

    def __str__(self) -> str:
        return str(self.value)

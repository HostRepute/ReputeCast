from enum import StrEnum

class ConnectProductionAccessInputLanguage(StrEnum):
    EN = "en"
    JA = "ja"

    def __str__(self) -> str:
        return str(self.value)

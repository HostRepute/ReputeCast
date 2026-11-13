from enum import StrEnum

class CountdownSettingsLanguage(StrEnum):
    DE = "de"
    EN = "en"
    ES = "es"
    FR = "fr"
    VI = "vi"

    def __str__(self) -> str:
        return str(self.value)

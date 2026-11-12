from enum import StrEnum

class ContactLanguageSource(StrEnum):
    API = "api"
    FORM = "form"
    IMPORT = "import"
    PREFERENCE_CENTER = "preference_center"
    TEAM = "team"
    VALUE_5 = ""

    def __str__(self) -> str:
        return str(self.value)

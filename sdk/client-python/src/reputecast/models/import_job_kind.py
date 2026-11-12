from enum import StrEnum

class ImportJobKind(StrEnum):
    EXPORT = "export"
    IMPORT = "import"
    OPTIN = "optin"
    VERIFICATION = "verification"

    def __str__(self) -> str:
        return str(self.value)

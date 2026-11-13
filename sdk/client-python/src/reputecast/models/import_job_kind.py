from enum import StrEnum

class ImportJobKind(StrEnum):
    EXPORT = "export"
    IMPORT = "import"
    OPTIN = "optin"
    POST_IMPORT = "post_import"
    VERIFICATION = "verification"

    def __str__(self) -> str:
        return str(self.value)

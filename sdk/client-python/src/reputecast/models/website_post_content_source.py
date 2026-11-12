from enum import StrEnum

class WebsitePostContentSource(StrEnum):
    AUTOSAVE = "autosave"
    IMPORT = "import"
    MANUAL = "manual"
    RESTORE = "restore"
    TEMPLATE = "template"
    VALUE_5 = ""

    def __str__(self) -> str:
        return str(self.value)

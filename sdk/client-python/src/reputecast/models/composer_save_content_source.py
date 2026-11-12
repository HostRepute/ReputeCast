from enum import StrEnum

class ComposerSaveContentSource(StrEnum):
    AUTOSAVE = "autosave"
    IMPORT = "import"
    MANUAL = "manual"
    RECOVERED = "recovered"
    TEMPLATE = "template"

    def __str__(self) -> str:
        return str(self.value)

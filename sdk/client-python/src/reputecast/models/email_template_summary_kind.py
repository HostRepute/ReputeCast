from enum import StrEnum

class EmailTemplateSummaryKind(StrEnum):
    SECTION = "section"
    TEMPLATE = "template"

    def __str__(self) -> str:
        return str(self.value)

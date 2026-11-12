from enum import StrEnum

class ContentCheckFixKind(StrEnum):
    EDIT_LINK = "edit_link"
    INSERT_FOOTER = "insert_footer"
    MINIFY = "minify"
    OPEN = "open"
    REMOVE_SECTION = "remove_section"
    REPLACE_LINK = "replace_link"
    SET_ALT = "set_alt"
    USE_SUBJECT = "use_subject"

    def __str__(self) -> str:
        return str(self.value)

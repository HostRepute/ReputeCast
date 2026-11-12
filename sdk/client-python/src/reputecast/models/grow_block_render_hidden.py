from enum import StrEnum

class GrowBlockRenderHidden(StrEnum):
    NONE = "none"
    NOT_ON_LIST = "not_on_list"
    NO_PROGRAM = "no_program"
    PROGRAM_OFF = "program_off"

    def __str__(self) -> str:
        return str(self.value)

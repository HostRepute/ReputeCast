from enum import StrEnum

class ListSmsRepliesKind(StrEnum):
    ALL = "all"
    HELP = "help"
    OPT_OUT = "opt_out"
    OTHER = "other"

    def __str__(self) -> str:
        return str(self.value)

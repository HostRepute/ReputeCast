from enum import StrEnum

class SmsReplyKind(StrEnum):
    HELP = "help"
    OPT_IN = "opt_in"
    OPT_OUT = "opt_out"
    REPLY = "reply"
    YES = "yes"

    def __str__(self) -> str:
        return str(self.value)

from enum import StrEnum

class LiveInDoubtRequestAction(StrEnum):
    LEAVE_OUT = "leave_out"
    RESEND = "resend"

    def __str__(self) -> str:
        return str(self.value)

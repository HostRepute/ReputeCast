from enum import StrEnum

class EarnMemberNext(StrEnum):
    ENDED = "ended"
    ENDS = "ends"
    NO_END = "no_end"
    RENEWS = "renews"
    RESUMES = "resumes"
    RETRIES = "retries"
    TRIAL_ENDS = "trial_ends"

    def __str__(self) -> str:
        return str(self.value)

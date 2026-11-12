from enum import StrEnum

class ReportProviderColumnComplaintRateNa(StrEnum):
    NOT_REPORTED = "not_reported"

    def __str__(self) -> str:
        return str(self.value)

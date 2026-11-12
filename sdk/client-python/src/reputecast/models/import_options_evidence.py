from enum import StrEnum

class ImportOptionsEvidence(StrEnum):
    REPERMISSION = "repermission"
    TRUST = "trust"

    def __str__(self) -> str:
        return str(self.value)

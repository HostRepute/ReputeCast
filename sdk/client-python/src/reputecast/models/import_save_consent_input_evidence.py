from enum import StrEnum

class ImportSaveConsentInputEvidence(StrEnum):
    REPERMISSION = "repermission"
    TRUST = "trust"

    def __str__(self) -> str:
        return str(self.value)

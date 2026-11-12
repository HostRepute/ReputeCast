from enum import StrEnum

class ImportConsentMappingEvidence(StrEnum):
    REPERMISSION = "repermission"
    TRUST = "trust"

    def __str__(self) -> str:
        return str(self.value)

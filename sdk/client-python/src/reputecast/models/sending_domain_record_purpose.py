from enum import StrEnum

class SendingDomainRecordPurpose(StrEnum):
    DKIM = "dkim"
    DMARC = "dmarc"
    RETURN_PATH = "return_path"
    SPF = "spf"
    VERIFICATION = "verification"

    def __str__(self) -> str:
        return str(self.value)

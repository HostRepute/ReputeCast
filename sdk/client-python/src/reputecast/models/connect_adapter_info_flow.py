from enum import StrEnum

class ConnectAdapterInfoFlow(StrEnum):
    API = "api"
    SES = "ses"
    SMTP = "smtp"

    def __str__(self) -> str:
        return str(self.value)

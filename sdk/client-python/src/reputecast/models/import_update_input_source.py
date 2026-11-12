from enum import StrEnum

class ImportUpdateInputSource(StrEnum):
    BREVO = "brevo"
    CSV = "csv"
    KEILA = "keila"
    LISTMONK = "listmonk"
    MAILCHIMP = "mailchimp"
    MAILERLITE = "mailerlite"
    NOTIFUSE = "notifuse"
    VALUE_8 = ""
    XLSX = "xlsx"

    def __str__(self) -> str:
        return str(self.value)

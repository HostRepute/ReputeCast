from enum import StrEnum

class ContactBulkInputAction(StrEnum):
    ADD_TAG = "add_tag"
    ADD_TO_LIST = "add_to_list"
    REMOVE_FROM_LIST = "remove_from_list"
    REMOVE_TAG = "remove_tag"
    RESEND_OPTIN = "resend_optin"
    SUPPRESS = "suppress"

    def __str__(self) -> str:
        return str(self.value)

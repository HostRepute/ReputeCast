from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_message_language_reason import TransactionalMessageLanguageReason
from ..types import UNSET, Unset






T = TypeVar("T", bound="TransactionalMessageLanguage")



@_attrs_define
class TransactionalMessageLanguage:
    """ 
        Attributes:
            sent (str):
            contact (str | Unset):
            reason (TransactionalMessageLanguageReason | Unset):
     """

    sent: str
    contact: str | Unset = UNSET
    reason: TransactionalMessageLanguageReason | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        sent = self.sent

        contact = self.contact

        reason: str | Unset = UNSET
        if not isinstance(self.reason, Unset):
            reason = self.reason.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "sent": sent,
        })
        if contact is not UNSET:
            field_dict["contact"] = contact
        if reason is not UNSET:
            field_dict["reason"] = reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        sent = d.pop("sent")

        contact = d.pop("contact", UNSET)

        _reason = d.pop("reason", UNSET)
        reason: TransactionalMessageLanguageReason | Unset
        if isinstance(_reason,  Unset):
            reason = UNSET
        else:
            reason = TransactionalMessageLanguageReason(_reason)




        transactional_message_language = cls(
            sent=sent,
            contact=contact,
            reason=reason,
        )

        return transactional_message_language


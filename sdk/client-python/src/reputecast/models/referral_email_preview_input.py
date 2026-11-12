from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from uuid import UUID






T = TypeVar("T", bound="ReferralEmailPreviewInput")



@_attrs_define
class ReferralEmailPreviewInput:
    """ 
        Attributes:
            contact_id (UUID | Unset):
            html (str | Unset):
            preheader (str | Unset):
            subject (str | Unset):
     """

    contact_id: UUID | Unset = UNSET
    html: str | Unset = UNSET
    preheader: str | Unset = UNSET
    subject: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        contact_id: str | Unset = UNSET
        if not isinstance(self.contact_id, Unset):
            contact_id = str(self.contact_id)

        html = self.html

        preheader = self.preheader

        subject = self.subject


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if contact_id is not UNSET:
            field_dict["contactId"] = contact_id
        if html is not UNSET:
            field_dict["html"] = html
        if preheader is not UNSET:
            field_dict["preheader"] = preheader
        if subject is not UNSET:
            field_dict["subject"] = subject

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        _contact_id = d.pop("contactId", UNSET)
        contact_id: UUID | Unset
        if isinstance(_contact_id,  Unset):
            contact_id = UNSET
        else:
            contact_id = UUID(_contact_id)




        html = d.pop("html", UNSET)

        preheader = d.pop("preheader", UNSET)

        subject = d.pop("subject", UNSET)

        referral_email_preview_input = cls(
            contact_id=contact_id,
            html=html,
            preheader=preheader,
            subject=subject,
        )

        return referral_email_preview_input


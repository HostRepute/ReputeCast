from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from uuid import UUID






T = TypeVar("T", bound="SmsPreviewInput")



@_attrs_define
class SmsPreviewInput:
    """ 
        Attributes:
            contact_id (UUID | Unset):
            country (str | Unset):
     """

    contact_id: UUID | Unset = UNSET
    country: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        contact_id: str | Unset = UNSET
        if not isinstance(self.contact_id, Unset):
            contact_id = str(self.contact_id)

        country = self.country


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if contact_id is not UNSET:
            field_dict["contactId"] = contact_id
        if country is not UNSET:
            field_dict["country"] = country

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




        country = d.pop("country", UNSET)

        sms_preview_input = cls(
            contact_id=contact_id,
            country=country,
        )

        return sms_preview_input


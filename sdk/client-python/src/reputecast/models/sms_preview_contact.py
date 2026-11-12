from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SmsPreviewContact")



@_attrs_define
class SmsPreviewContact:
    """ 
        Attributes:
            country (str):
            id (str):
            name (str):
     """

    country: str
    id: str
    name: str





    def to_dict(self) -> dict[str, Any]:
        country = self.country

        id = self.id

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "country": country,
            "id": id,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        country = d.pop("country")

        id = d.pop("id")

        name = d.pop("name")

        sms_preview_contact = cls(
            country=country,
            id=id,
            name=name,
        )

        return sms_preview_contact


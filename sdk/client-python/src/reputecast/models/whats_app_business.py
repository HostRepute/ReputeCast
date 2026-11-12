from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WhatsAppBusiness")



@_attrs_define
class WhatsAppBusiness:
    """ 
        Attributes:
            display_name (str):
            number (str):
     """

    display_name: str
    number: str





    def to_dict(self) -> dict[str, Any]:
        display_name = self.display_name

        number = self.number


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "displayName": display_name,
            "number": number,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        display_name = d.pop("displayName")

        number = d.pop("number")

        whats_app_business = cls(
            display_name=display_name,
            number=number,
        )

        return whats_app_business


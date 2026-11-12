from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PublicBrandingLogo")



@_attrs_define
class PublicBrandingLogo:
    """ 
        Attributes:
            name (str):  Example: acme-mark.svg.
            type_ (str):  Example: image/svg+xml.
            url (str):
     """

    name: str
    type_: str
    url: str





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        type_ = self.type_

        url = self.url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
            "type": type_,
            "url": url,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        type_ = d.pop("type")

        url = d.pop("url")

        public_branding_logo = cls(
            name=name,
            type_=type_,
            url=url,
        )

        return public_branding_logo


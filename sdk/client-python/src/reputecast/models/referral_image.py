from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReferralImage")



@_attrs_define
class ReferralImage:
    """ 
        Attributes:
            height (int):
            id (str):
            name (str):
            size (int):
            url (str):
            width (int):
     """

    height: int
    id: str
    name: str
    size: int
    url: str
    width: int





    def to_dict(self) -> dict[str, Any]:
        height = self.height

        id = self.id

        name = self.name

        size = self.size

        url = self.url

        width = self.width


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "height": height,
            "id": id,
            "name": name,
            "size": size,
            "url": url,
            "width": width,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        height = d.pop("height")

        id = d.pop("id")

        name = d.pop("name")

        size = d.pop("size")

        url = d.pop("url")

        width = d.pop("width")

        referral_image = cls(
            height=height,
            id=id,
            name=name,
            size=size,
            url=url,
            width=width,
        )

        return referral_image


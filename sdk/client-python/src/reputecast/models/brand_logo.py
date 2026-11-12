from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="BrandLogo")



@_attrs_define
class BrandLogo:
    """ 
        Attributes:
            file_name (str):
            height (int):
            media_id (str):
            url (str):
            width (int):
     """

    file_name: str
    height: int
    media_id: str
    url: str
    width: int





    def to_dict(self) -> dict[str, Any]:
        file_name = self.file_name

        height = self.height

        media_id = self.media_id

        url = self.url

        width = self.width


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "fileName": file_name,
            "height": height,
            "mediaId": media_id,
            "url": url,
            "width": width,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        file_name = d.pop("fileName")

        height = d.pop("height")

        media_id = d.pop("mediaId")

        url = d.pop("url")

        width = d.pop("width")

        brand_logo = cls(
            file_name=file_name,
            height=height,
            media_id=media_id,
            url=url,
            width=width,
        )

        return brand_logo


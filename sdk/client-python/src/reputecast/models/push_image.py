from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PushImage")



@_attrs_define
class PushImage:
    """ 
        Attributes:
            bytes_ (int):
            height (int):
            media_id (str):
            name (str):
            url (str):
            width (int):
     """

    bytes_: int
    height: int
    media_id: str
    name: str
    url: str
    width: int





    def to_dict(self) -> dict[str, Any]:
        bytes_ = self.bytes_

        height = self.height

        media_id = self.media_id

        name = self.name

        url = self.url

        width = self.width


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bytes": bytes_,
            "height": height,
            "mediaId": media_id,
            "name": name,
            "url": url,
            "width": width,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        bytes_ = d.pop("bytes")

        height = d.pop("height")

        media_id = d.pop("mediaId")

        name = d.pop("name")

        url = d.pop("url")

        width = d.pop("width")

        push_image = cls(
            bytes_=bytes_,
            height=height,
            media_id=media_id,
            name=name,
            url=url,
            width=width,
        )

        return push_image


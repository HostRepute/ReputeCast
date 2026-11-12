from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.media_original_content_type import MediaOriginalContentType






T = TypeVar("T", bound="MediaOriginal")



@_attrs_define
class MediaOriginal:
    """ 
        Attributes:
            content_type (MediaOriginalContentType):
            height (int):
            size_bytes (int):
            width (int):
     """

    content_type: MediaOriginalContentType
    height: int
    size_bytes: int
    width: int





    def to_dict(self) -> dict[str, Any]:
        content_type = self.content_type.value

        height = self.height

        size_bytes = self.size_bytes

        width = self.width


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contentType": content_type,
            "height": height,
            "sizeBytes": size_bytes,
            "width": width,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        content_type = MediaOriginalContentType(d.pop("contentType"))




        height = d.pop("height")

        size_bytes = d.pop("sizeBytes")

        width = d.pop("width")

        media_original = cls(
            content_type=content_type,
            height=height,
            size_bytes=size_bytes,
            width=width,
        )

        return media_original


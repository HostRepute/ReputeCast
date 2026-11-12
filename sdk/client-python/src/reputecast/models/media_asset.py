from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.media_asset_content_type import MediaAssetContentType
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.media_original import MediaOriginal





T = TypeVar("T", bound="MediaAsset")



@_attrs_define
class MediaAsset:
    """ 
        Attributes:
            content_type (MediaAssetContentType):
            created_at (datetime.datetime):
            created_by (str):
            file_name (str):
            has_alpha (bool): Transparent pixels (logos survive dark mode)
            height (int):
            id (str):
            size_bytes (int):
            url (str): Root-relative URL; emails get the absolute public URL
            width (int):
            original (MediaOriginal | Unset):
     """

    content_type: MediaAssetContentType
    created_at: datetime.datetime
    created_by: str
    file_name: str
    has_alpha: bool
    height: int
    id: str
    size_bytes: int
    url: str
    width: int
    original: MediaOriginal | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.media_original import MediaOriginal # noqa: PLC0415
        content_type = self.content_type.value

        created_at = self.created_at.isoformat()

        created_by = self.created_by

        file_name = self.file_name

        has_alpha = self.has_alpha

        height = self.height

        id = self.id

        size_bytes = self.size_bytes

        url = self.url

        width = self.width

        original: dict[str, Any] | Unset = UNSET
        if not isinstance(self.original, Unset):
            original = self.original.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contentType": content_type,
            "createdAt": created_at,
            "createdBy": created_by,
            "fileName": file_name,
            "hasAlpha": has_alpha,
            "height": height,
            "id": id,
            "sizeBytes": size_bytes,
            "url": url,
            "width": width,
        })
        if original is not UNSET:
            field_dict["original"] = original

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.media_original import MediaOriginal # noqa: PLC0415
        d = dict(src_dict)
        content_type = MediaAssetContentType(d.pop("contentType"))




        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        created_by = d.pop("createdBy")

        file_name = d.pop("fileName")

        has_alpha = d.pop("hasAlpha")

        height = d.pop("height")

        id = d.pop("id")

        size_bytes = d.pop("sizeBytes")

        url = d.pop("url")

        width = d.pop("width")

        _original = d.pop("original", UNSET)
        original: MediaOriginal | Unset
        if isinstance(_original,  Unset):
            original = UNSET
        else:
            original = MediaOriginal.from_dict(_original)




        media_asset = cls(
            content_type=content_type,
            created_at=created_at,
            created_by=created_by,
            file_name=file_name,
            has_alpha=has_alpha,
            height=height,
            id=id,
            size_bytes=size_bytes,
            url=url,
            width=width,
            original=original,
        )

        return media_asset


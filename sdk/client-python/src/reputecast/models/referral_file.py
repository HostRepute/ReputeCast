from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="ReferralFile")



@_attrs_define
class ReferralFile:
    """ 
        Attributes:
            content_type (str):
            downloads (int):
            id (str):
            name (str):
            size (int):
            uploaded_at (datetime.datetime):
            uploaded_by (str):
     """

    content_type: str
    downloads: int
    id: str
    name: str
    size: int
    uploaded_at: datetime.datetime
    uploaded_by: str





    def to_dict(self) -> dict[str, Any]:
        content_type = self.content_type

        downloads = self.downloads

        id = self.id

        name = self.name

        size = self.size

        uploaded_at = self.uploaded_at.isoformat()

        uploaded_by = self.uploaded_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contentType": content_type,
            "downloads": downloads,
            "id": id,
            "name": name,
            "size": size,
            "uploadedAt": uploaded_at,
            "uploadedBy": uploaded_by,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        content_type = d.pop("contentType")

        downloads = d.pop("downloads")

        id = d.pop("id")

        name = d.pop("name")

        size = d.pop("size")

        uploaded_at = datetime.datetime.fromisoformat(d.pop("uploadedAt"))




        uploaded_by = d.pop("uploadedBy")

        referral_file = cls(
            content_type=content_type,
            downloads=downloads,
            id=id,
            name=name,
            size=size,
            uploaded_at=uploaded_at,
            uploaded_by=uploaded_by,
        )

        return referral_file


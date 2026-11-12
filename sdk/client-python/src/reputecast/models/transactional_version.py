from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="TransactionalVersion")



@_attrs_define
class TransactionalVersion:
    """ 
        Attributes:
            avatar_color (str):
            change (str):
            live (bool):
            published_at (datetime.datetime):
            published_by (str):
            size_bytes (int):
            subject (str):
            version (int):
     """

    avatar_color: str
    change: str
    live: bool
    published_at: datetime.datetime
    published_by: str
    size_bytes: int
    subject: str
    version: int





    def to_dict(self) -> dict[str, Any]:
        avatar_color = self.avatar_color

        change = self.change

        live = self.live

        published_at = self.published_at.isoformat()

        published_by = self.published_by

        size_bytes = self.size_bytes

        subject = self.subject

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "avatarColor": avatar_color,
            "change": change,
            "live": live,
            "publishedAt": published_at,
            "publishedBy": published_by,
            "sizeBytes": size_bytes,
            "subject": subject,
            "version": version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        avatar_color = d.pop("avatarColor")

        change = d.pop("change")

        live = d.pop("live")

        published_at = datetime.datetime.fromisoformat(d.pop("publishedAt"))




        published_by = d.pop("publishedBy")

        size_bytes = d.pop("sizeBytes")

        subject = d.pop("subject")

        version = d.pop("version")

        transactional_version = cls(
            avatar_color=avatar_color,
            change=change,
            live=live,
            published_at=published_at,
            published_by=published_by,
            size_bytes=size_bytes,
            subject=subject,
            version=version,
        )

        return transactional_version


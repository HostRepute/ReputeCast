from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="WebsitePostSaved")



@_attrs_define
class WebsitePostSaved:
    """ 
        Attributes:
            saved_at (datetime.datetime):
            version (int):
     """

    saved_at: datetime.datetime
    version: int





    def to_dict(self) -> dict[str, Any]:
        saved_at = self.saved_at.isoformat()

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "savedAt": saved_at,
            "version": version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        saved_at = datetime.datetime.fromisoformat(d.pop("savedAt"))




        version = d.pop("version")

        website_post_saved = cls(
            saved_at=saved_at,
            version=version,
        )

        return website_post_saved


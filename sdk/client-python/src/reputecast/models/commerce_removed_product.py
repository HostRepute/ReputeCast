from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="CommerceRemovedProduct")



@_attrs_define
class CommerceRemovedProduct:
    """ 
        Attributes:
            id (str):
            platform (str):
            title (str):
            removed_at (datetime.datetime | Unset):
     """

    id: str
    platform: str
    title: str
    removed_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        platform = self.platform

        title = self.title

        removed_at: str | Unset = UNSET
        if not isinstance(self.removed_at, Unset):
            removed_at = self.removed_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "platform": platform,
            "title": title,
        })
        if removed_at is not UNSET:
            field_dict["removedAt"] = removed_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        platform = d.pop("platform")

        title = d.pop("title")

        _removed_at = d.pop("removedAt", UNSET)
        removed_at: datetime.datetime | Unset
        if isinstance(_removed_at,  Unset):
            removed_at = UNSET
        else:
            removed_at = datetime.datetime.fromisoformat(_removed_at)




        commerce_removed_product = cls(
            id=id,
            platform=platform,
            title=title,
            removed_at=removed_at,
        )

        return commerce_removed_product


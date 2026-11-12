from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.composer_version import ComposerVersion





T = TypeVar("T", bound="ComposerVersionList")



@_attrs_define
class ComposerVersionList:
    """ 
        Attributes:
            items (list[ComposerVersion]):
            named (int):
            total (int):
            next_before (int | Unset):
     """

    items: list[ComposerVersion]
    named: int
    total: int
    next_before: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.composer_version import ComposerVersion # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        named = self.named

        total = self.total

        next_before = self.next_before


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "items": items,
            "named": named,
            "total": total,
        })
        if next_before is not UNSET:
            field_dict["nextBefore"] = next_before

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.composer_version import ComposerVersion # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = ComposerVersion.from_dict(items_item_data)



            items.append(items_item)


        named = d.pop("named")

        total = d.pop("total")

        next_before = d.pop("nextBefore", UNSET)

        composer_version_list = cls(
            items=items,
            named=named,
            total=total,
            next_before=next_before,
        )

        return composer_version_list


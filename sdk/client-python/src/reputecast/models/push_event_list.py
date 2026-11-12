from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.push_event import PushEvent





T = TypeVar("T", bound="PushEventList")



@_attrs_define
class PushEventList:
    """ 
        Attributes:
            items (list[PushEvent]):
            more (bool):
            removed (int):
     """

    items: list[PushEvent]
    more: bool
    removed: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.push_event import PushEvent # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        more = self.more

        removed = self.removed


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "items": items,
            "more": more,
            "removed": removed,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.push_event import PushEvent # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = PushEvent.from_dict(items_item_data)



            items.append(items_item)


        more = d.pop("more")

        removed = d.pop("removed")

        push_event_list = cls(
            items=items,
            more=more,
            removed=removed,
        )

        return push_event_list


from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.commerce_activity_item import CommerceActivityItem





T = TypeVar("T", bound="CommerceActivityList")



@_attrs_define
class CommerceActivityList:
    """ 
        Attributes:
            items (list[CommerceActivityItem]):
            next_ (str | Unset):
     """

    items: list[CommerceActivityItem]
    next_: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.commerce_activity_item import CommerceActivityItem # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        next_ = self.next_


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "items": items,
        })
        if next_ is not UNSET:
            field_dict["next"] = next_

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.commerce_activity_item import CommerceActivityItem # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = CommerceActivityItem.from_dict(items_item_data)



            items.append(items_item)


        next_ = d.pop("next", UNSET)

        commerce_activity_list = cls(
            items=items,
            next_=next_,
        )

        return commerce_activity_list


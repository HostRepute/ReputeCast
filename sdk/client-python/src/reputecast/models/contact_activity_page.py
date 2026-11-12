from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.contact_activity_item import ContactActivityItem





T = TypeVar("T", bound="ContactActivityPage")



@_attrs_define
class ContactActivityPage:
    """ 
        Attributes:
            items (list[ContactActivityItem]):
            total (int): Events matching the filter
            next_cursor (str | Unset):
     """

    items: list[ContactActivityItem]
    total: int
    next_cursor: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.contact_activity_item import ContactActivityItem # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        total = self.total

        next_cursor = self.next_cursor


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "items": items,
            "total": total,
        })
        if next_cursor is not UNSET:
            field_dict["nextCursor"] = next_cursor

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.contact_activity_item import ContactActivityItem # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = ContactActivityItem.from_dict(items_item_data)



            items.append(items_item)


        total = d.pop("total")

        next_cursor = d.pop("nextCursor", UNSET)

        contact_activity_page = cls(
            items=items,
            total=total,
            next_cursor=next_cursor,
        )

        return contact_activity_page


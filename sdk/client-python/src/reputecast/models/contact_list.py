from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.contact_list_item import ContactListItem





T = TypeVar("T", bound="ContactList")



@_attrs_define
class ContactList:
    """ 
        Attributes:
            items (list[ContactListItem]):
            page (int):
            page_size (int):
            total (int):
            next_cursor (str | Unset):
            prev_cursor (str | Unset):
            total_capped (bool | Unset):
     """

    items: list[ContactListItem]
    page: int
    page_size: int
    total: int
    next_cursor: str | Unset = UNSET
    prev_cursor: str | Unset = UNSET
    total_capped: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.contact_list_item import ContactListItem # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        page = self.page

        page_size = self.page_size

        total = self.total

        next_cursor = self.next_cursor

        prev_cursor = self.prev_cursor

        total_capped = self.total_capped


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "items": items,
            "page": page,
            "pageSize": page_size,
            "total": total,
        })
        if next_cursor is not UNSET:
            field_dict["nextCursor"] = next_cursor
        if prev_cursor is not UNSET:
            field_dict["prevCursor"] = prev_cursor
        if total_capped is not UNSET:
            field_dict["totalCapped"] = total_capped

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.contact_list_item import ContactListItem # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = ContactListItem.from_dict(items_item_data)



            items.append(items_item)


        page = d.pop("page")

        page_size = d.pop("pageSize")

        total = d.pop("total")

        next_cursor = d.pop("nextCursor", UNSET)

        prev_cursor = d.pop("prevCursor", UNSET)

        total_capped = d.pop("totalCapped", UNSET)

        contact_list = cls(
            items=items,
            page=page,
            page_size=page_size,
            total=total,
            next_cursor=next_cursor,
            prev_cursor=prev_cursor,
            total_capped=total_capped,
        )

        return contact_list


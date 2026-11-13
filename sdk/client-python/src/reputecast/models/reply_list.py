from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.reply_counts import ReplyCounts
  from ..models.reply_list_item import ReplyListItem





T = TypeVar("T", bound="ReplyList")



@_attrs_define
class ReplyList:
    """ 
        Attributes:
            can_manage (bool):
            counts (ReplyCounts):
            items (list[ReplyListItem]):
            receiving (bool):
            domain (str | Unset):
            next_cursor (str | Unset):
     """

    can_manage: bool
    counts: ReplyCounts
    items: list[ReplyListItem]
    receiving: bool
    domain: str | Unset = UNSET
    next_cursor: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.reply_counts import ReplyCounts # noqa: PLC0415
        from ..models.reply_list_item import ReplyListItem # noqa: PLC0415
        can_manage = self.can_manage

        counts = self.counts.to_dict()

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        receiving = self.receiving

        domain = self.domain

        next_cursor = self.next_cursor


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canManage": can_manage,
            "counts": counts,
            "items": items,
            "receiving": receiving,
        })
        if domain is not UNSET:
            field_dict["domain"] = domain
        if next_cursor is not UNSET:
            field_dict["nextCursor"] = next_cursor

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.reply_counts import ReplyCounts # noqa: PLC0415
        from ..models.reply_list_item import ReplyListItem # noqa: PLC0415
        d = dict(src_dict)
        can_manage = d.pop("canManage")

        counts = ReplyCounts.from_dict(d.pop("counts"))




        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = ReplyListItem.from_dict(items_item_data)



            items.append(items_item)


        receiving = d.pop("receiving")

        domain = d.pop("domain", UNSET)

        next_cursor = d.pop("nextCursor", UNSET)

        reply_list = cls(
            can_manage=can_manage,
            counts=counts,
            items=items,
            receiving=receiving,
            domain=domain,
            next_cursor=next_cursor,
        )

        return reply_list


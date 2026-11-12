from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.gift_counts import GiftCounts
  from ..models.gift_row import GiftRow





T = TypeVar("T", bound="GiftList")



@_attrs_define
class GiftList:
    """ 
        Attributes:
            counts (GiftCounts):
            items (list[GiftRow]):
            page (int):
            page_size (int):
            status (str):
            total (int):
     """

    counts: GiftCounts
    items: list[GiftRow]
    page: int
    page_size: int
    status: str
    total: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.gift_counts import GiftCounts # noqa: PLC0415
        from ..models.gift_row import GiftRow # noqa: PLC0415
        counts = self.counts.to_dict()

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        page = self.page

        page_size = self.page_size

        status = self.status

        total = self.total


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "counts": counts,
            "items": items,
            "page": page,
            "pageSize": page_size,
            "status": status,
            "total": total,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.gift_counts import GiftCounts # noqa: PLC0415
        from ..models.gift_row import GiftRow # noqa: PLC0415
        d = dict(src_dict)
        counts = GiftCounts.from_dict(d.pop("counts"))




        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = GiftRow.from_dict(items_item_data)



            items.append(items_item)


        page = d.pop("page")

        page_size = d.pop("pageSize")

        status = d.pop("status")

        total = d.pop("total")

        gift_list = cls(
            counts=counts,
            items=items,
            page=page,
            page_size=page_size,
            status=status,
            total=total,
        )

        return gift_list


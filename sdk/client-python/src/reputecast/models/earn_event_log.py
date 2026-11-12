from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.earn_event import EarnEvent





T = TypeVar("T", bound="EarnEventLog")



@_attrs_define
class EarnEventLog:
    """ 
        Attributes:
            items (list[EarnEvent]):
            page (int):
            page_size (int):
            range_ (str):
            total (int):
            types (list[str]):
     """

    items: list[EarnEvent]
    page: int
    page_size: int
    range_: str
    total: int
    types: list[str]





    def to_dict(self) -> dict[str, Any]:
        from ..models.earn_event import EarnEvent # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        page = self.page

        page_size = self.page_size

        range_ = self.range_

        total = self.total

        types = self.types




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "items": items,
            "page": page,
            "pageSize": page_size,
            "range": range_,
            "total": total,
            "types": types,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.earn_event import EarnEvent # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = EarnEvent.from_dict(items_item_data)



            items.append(items_item)


        page = d.pop("page")

        page_size = d.pop("pageSize")

        range_ = d.pop("range")

        total = d.pop("total")

        types = cast(list[str], d.pop("types"))


        earn_event_log = cls(
            items=items,
            page=page,
            page_size=page_size,
            range_=range_,
            total=total,
            types=types,
        )

        return earn_event_log


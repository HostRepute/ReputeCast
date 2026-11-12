from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.conversion_item import ConversionItem





T = TypeVar("T", bound="ConversionList")



@_attrs_define
class ConversionList:
    """ 
        Attributes:
            items (list[ConversionItem]):
            window (int):
            next_ (datetime.datetime | Unset):
     """

    items: list[ConversionItem]
    window: int
    next_: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.conversion_item import ConversionItem # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        window = self.window

        next_: str | Unset = UNSET
        if not isinstance(self.next_, Unset):
            next_ = self.next_.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "items": items,
            "window": window,
        })
        if next_ is not UNSET:
            field_dict["next"] = next_

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.conversion_item import ConversionItem # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = ConversionItem.from_dict(items_item_data)



            items.append(items_item)


        window = d.pop("window")

        _next_ = d.pop("next", UNSET)
        next_: datetime.datetime | Unset
        if isinstance(_next_,  Unset):
            next_ = UNSET
        else:
            next_ = datetime.datetime.fromisoformat(_next_)




        conversion_list = cls(
            items=items,
            window=window,
            next_=next_,
        )

        return conversion_list


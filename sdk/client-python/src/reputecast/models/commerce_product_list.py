from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.commerce_product import CommerceProduct





T = TypeVar("T", bound="CommerceProductList")



@_attrs_define
class CommerceProductList:
    """ 
        Attributes:
            items (list[CommerceProduct]):
            total (int):
            next_ (int | Unset):
     """

    items: list[CommerceProduct]
    total: int
    next_: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.commerce_product import CommerceProduct # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        total = self.total

        next_ = self.next_


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "items": items,
            "total": total,
        })
        if next_ is not UNSET:
            field_dict["next"] = next_

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.commerce_product import CommerceProduct # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = CommerceProduct.from_dict(items_item_data)



            items.append(items_item)


        total = d.pop("total")

        next_ = d.pop("next", UNSET)

        commerce_product_list = cls(
            items=items,
            total=total,
            next_=next_,
        )

        return commerce_product_list


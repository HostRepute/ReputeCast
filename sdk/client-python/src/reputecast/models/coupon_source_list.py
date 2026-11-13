from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.coupon_source import CouponSource





T = TypeVar("T", bound="CouponSourceList")



@_attrs_define
class CouponSourceList:
    """ 
        Attributes:
            items (list[CouponSource]):
     """

    items: list[CouponSource]





    def to_dict(self) -> dict[str, Any]:
        from ..models.coupon_source import CouponSource # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "items": items,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.coupon_source import CouponSource # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = CouponSource.from_dict(items_item_data)



            items.append(items_item)


        coupon_source_list = cls(
            items=items,
        )

        return coupon_source_list


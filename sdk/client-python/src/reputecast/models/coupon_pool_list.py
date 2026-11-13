from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.coupon_pool_list_item import CouponPoolListItem





T = TypeVar("T", bound="CouponPoolList")



@_attrs_define
class CouponPoolList:
    """ 
        Attributes:
            helpers (list[str]):
            items (list[CouponPoolListItem]):
     """

    helpers: list[str]
    items: list[CouponPoolListItem]





    def to_dict(self) -> dict[str, Any]:
        from ..models.coupon_pool_list_item import CouponPoolListItem # noqa: PLC0415
        helpers = self.helpers



        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "helpers": helpers,
            "items": items,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.coupon_pool_list_item import CouponPoolListItem # noqa: PLC0415
        d = dict(src_dict)
        helpers = cast(list[str], d.pop("helpers"))


        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = CouponPoolListItem.from_dict(items_item_data)



            items.append(items_item)


        coupon_pool_list = cls(
            helpers=helpers,
            items=items,
        )

        return coupon_pool_list


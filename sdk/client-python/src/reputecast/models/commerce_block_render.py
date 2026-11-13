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
  from ..models.commerce_removed_product import CommerceRemovedProduct





T = TypeVar("T", bound="CommerceBlockRender")



@_attrs_define
class CommerceBlockRender:
    """ 
        Attributes:
            disconnected (bool):
            html (str):
            platform (str):
            products (list[CommerceProduct]):
            removed (list[CommerceRemovedProduct]):
            sold_out (list[str]):
            store_name (str):
            note (str | Unset):
     """

    disconnected: bool
    html: str
    platform: str
    products: list[CommerceProduct]
    removed: list[CommerceRemovedProduct]
    sold_out: list[str]
    store_name: str
    note: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.commerce_product import CommerceProduct # noqa: PLC0415
        from ..models.commerce_removed_product import CommerceRemovedProduct # noqa: PLC0415
        disconnected = self.disconnected

        html = self.html

        platform = self.platform

        products = []
        for products_item_data in self.products:
            products_item = products_item_data.to_dict()
            products.append(products_item)



        removed = []
        for removed_item_data in self.removed:
            removed_item = removed_item_data.to_dict()
            removed.append(removed_item)



        sold_out = self.sold_out



        store_name = self.store_name

        note = self.note


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "disconnected": disconnected,
            "html": html,
            "platform": platform,
            "products": products,
            "removed": removed,
            "soldOut": sold_out,
            "storeName": store_name,
        })
        if note is not UNSET:
            field_dict["note"] = note

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.commerce_product import CommerceProduct # noqa: PLC0415
        from ..models.commerce_removed_product import CommerceRemovedProduct # noqa: PLC0415
        d = dict(src_dict)
        disconnected = d.pop("disconnected")

        html = d.pop("html")

        platform = d.pop("platform")

        products = []
        _products = d.pop("products")
        for products_item_data in (_products):
            products_item = CommerceProduct.from_dict(products_item_data)



            products.append(products_item)


        removed = []
        _removed = d.pop("removed")
        for removed_item_data in (_removed):
            removed_item = CommerceRemovedProduct.from_dict(removed_item_data)



            removed.append(removed_item)


        sold_out = cast(list[str], d.pop("soldOut"))


        store_name = d.pop("storeName")

        note = d.pop("note", UNSET)

        commerce_block_render = cls(
            disconnected=disconnected,
            html=html,
            platform=platform,
            products=products,
            removed=removed,
            sold_out=sold_out,
            store_name=store_name,
            note=note,
        )

        return commerce_block_render


from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.product_count import ProductCount
  from ..models.product_order_item import ProductOrderItem





T = TypeVar("T", bound="ProductOrderList")



@_attrs_define
class ProductOrderList:
    """ 
        Attributes:
            items (list[ProductOrderItem]):
            page (int):
            page_size (int):
            products (list[ProductCount]):
            total (int):
     """

    items: list[ProductOrderItem]
    page: int
    page_size: int
    products: list[ProductCount]
    total: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.product_count import ProductCount # noqa: PLC0415
        from ..models.product_order_item import ProductOrderItem # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        page = self.page

        page_size = self.page_size

        products = []
        for products_item_data in self.products:
            products_item = products_item_data.to_dict()
            products.append(products_item)



        total = self.total


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "items": items,
            "page": page,
            "pageSize": page_size,
            "products": products,
            "total": total,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.product_count import ProductCount # noqa: PLC0415
        from ..models.product_order_item import ProductOrderItem # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = ProductOrderItem.from_dict(items_item_data)



            items.append(items_item)


        page = d.pop("page")

        page_size = d.pop("pageSize")

        products = []
        _products = d.pop("products")
        for products_item_data in (_products):
            products_item = ProductCount.from_dict(products_item_data)



            products.append(products_item)


        total = d.pop("total")

        product_order_list = cls(
            items=items,
            page=page,
            page_size=page_size,
            products=products,
            total=total,
        )

        return product_order_list


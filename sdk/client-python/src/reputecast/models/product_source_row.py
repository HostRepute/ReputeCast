from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.product_source_row_source import ProductSourceRowSource






T = TypeVar("T", bound="ProductSourceRow")



@_attrs_define
class ProductSourceRow:
    """ 
        Attributes:
            gross_cents (int):
            orders (int):
            product (str):
            product_id (str):
            source (ProductSourceRowSource):
     """

    gross_cents: int
    orders: int
    product: str
    product_id: str
    source: ProductSourceRowSource





    def to_dict(self) -> dict[str, Any]:
        gross_cents = self.gross_cents

        orders = self.orders

        product = self.product

        product_id = self.product_id

        source = self.source.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "grossCents": gross_cents,
            "orders": orders,
            "product": product,
            "productId": product_id,
            "source": source,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        gross_cents = d.pop("grossCents")

        orders = d.pop("orders")

        product = d.pop("product")

        product_id = d.pop("productId")

        source = ProductSourceRowSource(d.pop("source"))




        product_source_row = cls(
            gross_cents=gross_cents,
            orders=orders,
            product=product,
            product_id=product_id,
            source=source,
        )

        return product_source_row


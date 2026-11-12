from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.product_count import ProductCount





T = TypeVar("T", bound="ProductsStats")



@_attrs_define
class ProductsStats:
    """ 
        Attributes:
            by_product (list[ProductCount]):
            gross_cents (int):
            net_cents (int):
            open_disputes (int):
            orders (int):
            refunded_cents (int):
            refunded_orders (int):
     """

    by_product: list[ProductCount]
    gross_cents: int
    net_cents: int
    open_disputes: int
    orders: int
    refunded_cents: int
    refunded_orders: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.product_count import ProductCount # noqa: PLC0415
        by_product = []
        for by_product_item_data in self.by_product:
            by_product_item = by_product_item_data.to_dict()
            by_product.append(by_product_item)



        gross_cents = self.gross_cents

        net_cents = self.net_cents

        open_disputes = self.open_disputes

        orders = self.orders

        refunded_cents = self.refunded_cents

        refunded_orders = self.refunded_orders


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "byProduct": by_product,
            "grossCents": gross_cents,
            "netCents": net_cents,
            "openDisputes": open_disputes,
            "orders": orders,
            "refundedCents": refunded_cents,
            "refundedOrders": refunded_orders,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.product_count import ProductCount # noqa: PLC0415
        d = dict(src_dict)
        by_product = []
        _by_product = d.pop("byProduct")
        for by_product_item_data in (_by_product):
            by_product_item = ProductCount.from_dict(by_product_item_data)



            by_product.append(by_product_item)


        gross_cents = d.pop("grossCents")

        net_cents = d.pop("netCents")

        open_disputes = d.pop("openDisputes")

        orders = d.pop("orders")

        refunded_cents = d.pop("refundedCents")

        refunded_orders = d.pop("refundedOrders")

        products_stats = cls(
            by_product=by_product,
            gross_cents=gross_cents,
            net_cents=net_cents,
            open_disputes=open_disputes,
            orders=orders,
            refunded_cents=refunded_cents,
            refunded_orders=refunded_orders,
        )

        return products_stats


from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="ProductsSummary")



@_attrs_define
class ProductsSummary:
    """ 
        Attributes:
            currency (str):
            from_ (datetime.datetime):
            gross_cents (int):
            orders (int):
            products (int):
            refunded_cents (int):
            refunded_orders (int):
            until (datetime.datetime):
     """

    currency: str
    from_: datetime.datetime
    gross_cents: int
    orders: int
    products: int
    refunded_cents: int
    refunded_orders: int
    until: datetime.datetime





    def to_dict(self) -> dict[str, Any]:
        currency = self.currency

        from_ = self.from_.isoformat()

        gross_cents = self.gross_cents

        orders = self.orders

        products = self.products

        refunded_cents = self.refunded_cents

        refunded_orders = self.refunded_orders

        until = self.until.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "currency": currency,
            "from": from_,
            "grossCents": gross_cents,
            "orders": orders,
            "products": products,
            "refundedCents": refunded_cents,
            "refundedOrders": refunded_orders,
            "until": until,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        currency = d.pop("currency")

        from_ = datetime.datetime.fromisoformat(d.pop("from"))




        gross_cents = d.pop("grossCents")

        orders = d.pop("orders")

        products = d.pop("products")

        refunded_cents = d.pop("refundedCents")

        refunded_orders = d.pop("refundedOrders")

        until = datetime.datetime.fromisoformat(d.pop("until"))




        products_summary = cls(
            currency=currency,
            from_=from_,
            gross_cents=gross_cents,
            orders=orders,
            products=products,
            refunded_cents=refunded_cents,
            refunded_orders=refunded_orders,
            until=until,
        )

        return products_summary


from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="CommerceRevenueSummary")



@_attrs_define
class CommerceRevenueSummary:
    """ 
        Attributes:
            credited (int):
            credited_revenue (int):
            currency (str):
            days (int):
            orders (int):
            revenue (int):
            share (float):
            stores (int):
     """

    credited: int
    credited_revenue: int
    currency: str
    days: int
    orders: int
    revenue: int
    share: float
    stores: int





    def to_dict(self) -> dict[str, Any]:
        credited = self.credited

        credited_revenue = self.credited_revenue

        currency = self.currency

        days = self.days

        orders = self.orders

        revenue = self.revenue

        share = self.share

        stores = self.stores


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "credited": credited,
            "creditedRevenue": credited_revenue,
            "currency": currency,
            "days": days,
            "orders": orders,
            "revenue": revenue,
            "share": share,
            "stores": stores,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        credited = d.pop("credited")

        credited_revenue = d.pop("creditedRevenue")

        currency = d.pop("currency")

        days = d.pop("days")

        orders = d.pop("orders")

        revenue = d.pop("revenue")

        share = d.pop("share")

        stores = d.pop("stores")

        commerce_revenue_summary = cls(
            credited=credited,
            credited_revenue=credited_revenue,
            currency=currency,
            days=days,
            orders=orders,
            revenue=revenue,
            share=share,
            stores=stores,
        )

        return commerce_revenue_summary


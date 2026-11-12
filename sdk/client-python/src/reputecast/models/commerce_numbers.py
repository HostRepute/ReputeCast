from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="CommerceNumbers")



@_attrs_define
class CommerceNumbers:
    """ 
        Attributes:
            credited (int):
            credited_revenue (int):
            currency (str):
            days (int):
            list_members (int):
            orders (int):
            revenue (int):
            share (float):
     """

    credited: int
    credited_revenue: int
    currency: str
    days: int
    list_members: int
    orders: int
    revenue: int
    share: float





    def to_dict(self) -> dict[str, Any]:
        credited = self.credited

        credited_revenue = self.credited_revenue

        currency = self.currency

        days = self.days

        list_members = self.list_members

        orders = self.orders

        revenue = self.revenue

        share = self.share


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "credited": credited,
            "creditedRevenue": credited_revenue,
            "currency": currency,
            "days": days,
            "listMembers": list_members,
            "orders": orders,
            "revenue": revenue,
            "share": share,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        credited = d.pop("credited")

        credited_revenue = d.pop("creditedRevenue")

        currency = d.pop("currency")

        days = d.pop("days")

        list_members = d.pop("listMembers")

        orders = d.pop("orders")

        revenue = d.pop("revenue")

        share = d.pop("share")

        commerce_numbers = cls(
            credited=credited,
            credited_revenue=credited_revenue,
            currency=currency,
            days=days,
            list_members=list_members,
            orders=orders,
            revenue=revenue,
            share=share,
        )

        return commerce_numbers


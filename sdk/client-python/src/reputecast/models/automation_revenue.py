from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="AutomationRevenue")



@_attrs_define
class AutomationRevenue:
    """ 
        Attributes:
            currency (str):
            orders (int):
            revenue (int): In minor units of the currency
     """

    currency: str
    orders: int
    revenue: int





    def to_dict(self) -> dict[str, Any]:
        currency = self.currency

        orders = self.orders

        revenue = self.revenue


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "currency": currency,
            "orders": orders,
            "revenue": revenue,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        currency = d.pop("currency")

        orders = d.pop("orders")

        revenue = d.pop("revenue")

        automation_revenue = cls(
            currency=currency,
            orders=orders,
            revenue=revenue,
        )

        return automation_revenue


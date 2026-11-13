from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="HoldoutGroupResult")



@_attrs_define
class HoldoutGroupResult:
    """ 
        Attributes:
            contacts (int):
            converters (int):
            orders (int):
            revenue_cents (int):
            visitors (int):
     """

    contacts: int
    converters: int
    orders: int
    revenue_cents: int
    visitors: int





    def to_dict(self) -> dict[str, Any]:
        contacts = self.contacts

        converters = self.converters

        orders = self.orders

        revenue_cents = self.revenue_cents

        visitors = self.visitors


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contacts": contacts,
            "converters": converters,
            "orders": orders,
            "revenueCents": revenue_cents,
            "visitors": visitors,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        contacts = d.pop("contacts")

        converters = d.pop("converters")

        orders = d.pop("orders")

        revenue_cents = d.pop("revenueCents")

        visitors = d.pop("visitors")

        holdout_group_result = cls(
            contacts=contacts,
            converters=converters,
            orders=orders,
            revenue_cents=revenue_cents,
            visitors=visitors,
        )

        return holdout_group_result


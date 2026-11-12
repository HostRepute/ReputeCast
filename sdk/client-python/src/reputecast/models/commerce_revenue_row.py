from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.commerce_revenue_row_kind import CommerceRevenueRowKind
from ..types import UNSET, Unset






T = TypeVar("T", bound="CommerceRevenueRow")



@_attrs_define
class CommerceRevenueRow:
    """ 
        Attributes:
            count (int):
            kind (CommerceRevenueRowKind):
            name (str):
            orders (int):
            revenue (int):
            id (str | Unset):
     """

    count: int
    kind: CommerceRevenueRowKind
    name: str
    orders: int
    revenue: int
    id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        kind = self.kind.value

        name = self.name

        orders = self.orders

        revenue = self.revenue

        id = self.id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "kind": kind,
            "name": name,
            "orders": orders,
            "revenue": revenue,
        })
        if id is not UNSET:
            field_dict["id"] = id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        kind = CommerceRevenueRowKind(d.pop("kind"))




        name = d.pop("name")

        orders = d.pop("orders")

        revenue = d.pop("revenue")

        id = d.pop("id", UNSET)

        commerce_revenue_row = cls(
            count=count,
            kind=kind,
            name=name,
            orders=orders,
            revenue=revenue,
            id=id,
        )

        return commerce_revenue_row


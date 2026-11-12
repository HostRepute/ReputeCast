from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="CommerceRecoveredOrders")



@_attrs_define
class CommerceRecoveredOrders:
    """ 
        Attributes:
            orders (int):
            revenue (int):
            since (datetime.datetime | Unset):
     """

    orders: int
    revenue: int
    since: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        orders = self.orders

        revenue = self.revenue

        since: str | Unset = UNSET
        if not isinstance(self.since, Unset):
            since = self.since.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "orders": orders,
            "revenue": revenue,
        })
        if since is not UNSET:
            field_dict["since"] = since

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        orders = d.pop("orders")

        revenue = d.pop("revenue")

        _since = d.pop("since", UNSET)
        since: datetime.datetime | Unset
        if isinstance(_since,  Unset):
            since = UNSET
        else:
            since = datetime.datetime.fromisoformat(_since)




        commerce_recovered_orders = cls(
            orders=orders,
            revenue=revenue,
            since=since,
        )

        return commerce_recovered_orders


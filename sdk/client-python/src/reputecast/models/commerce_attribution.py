from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.commerce_revenue_row import CommerceRevenueRow





T = TypeVar("T", bound="CommerceAttribution")



@_attrs_define
class CommerceAttribution:
    """ 
        Attributes:
            by_email (int):
            by_token (int):
            credited (int):
            credited_revenue (int):
            days (int):
            orders (int):
            param (str):
            revenue (int):
            rows (list[CommerceRevenueRow]):
            tag_example (str):
            tag_host (str):
            tag_links (bool):
            others (CommerceRevenueRow | Unset):
            set_at (datetime.datetime | Unset):
            set_by (str | Unset):
     """

    by_email: int
    by_token: int
    credited: int
    credited_revenue: int
    days: int
    orders: int
    param: str
    revenue: int
    rows: list[CommerceRevenueRow]
    tag_example: str
    tag_host: str
    tag_links: bool
    others: CommerceRevenueRow | Unset = UNSET
    set_at: datetime.datetime | Unset = UNSET
    set_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.commerce_revenue_row import CommerceRevenueRow # noqa: PLC0415
        by_email = self.by_email

        by_token = self.by_token

        credited = self.credited

        credited_revenue = self.credited_revenue

        days = self.days

        orders = self.orders

        param = self.param

        revenue = self.revenue

        rows = []
        for rows_item_data in self.rows:
            rows_item = rows_item_data.to_dict()
            rows.append(rows_item)



        tag_example = self.tag_example

        tag_host = self.tag_host

        tag_links = self.tag_links

        others: dict[str, Any] | Unset = UNSET
        if not isinstance(self.others, Unset):
            others = self.others.to_dict()

        set_at: str | Unset = UNSET
        if not isinstance(self.set_at, Unset):
            set_at = self.set_at.isoformat()

        set_by = self.set_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "byEmail": by_email,
            "byToken": by_token,
            "credited": credited,
            "creditedRevenue": credited_revenue,
            "days": days,
            "orders": orders,
            "param": param,
            "revenue": revenue,
            "rows": rows,
            "tagExample": tag_example,
            "tagHost": tag_host,
            "tagLinks": tag_links,
        })
        if others is not UNSET:
            field_dict["others"] = others
        if set_at is not UNSET:
            field_dict["setAt"] = set_at
        if set_by is not UNSET:
            field_dict["setBy"] = set_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.commerce_revenue_row import CommerceRevenueRow # noqa: PLC0415
        d = dict(src_dict)
        by_email = d.pop("byEmail")

        by_token = d.pop("byToken")

        credited = d.pop("credited")

        credited_revenue = d.pop("creditedRevenue")

        days = d.pop("days")

        orders = d.pop("orders")

        param = d.pop("param")

        revenue = d.pop("revenue")

        rows = []
        _rows = d.pop("rows")
        for rows_item_data in (_rows):
            rows_item = CommerceRevenueRow.from_dict(rows_item_data)



            rows.append(rows_item)


        tag_example = d.pop("tagExample")

        tag_host = d.pop("tagHost")

        tag_links = d.pop("tagLinks")

        _others = d.pop("others", UNSET)
        others: CommerceRevenueRow | Unset
        if isinstance(_others,  Unset):
            others = UNSET
        else:
            others = CommerceRevenueRow.from_dict(_others)




        _set_at = d.pop("setAt", UNSET)
        set_at: datetime.datetime | Unset
        if isinstance(_set_at,  Unset):
            set_at = UNSET
        else:
            set_at = datetime.datetime.fromisoformat(_set_at)




        set_by = d.pop("setBy", UNSET)

        commerce_attribution = cls(
            by_email=by_email,
            by_token=by_token,
            credited=credited,
            credited_revenue=credited_revenue,
            days=days,
            orders=orders,
            param=param,
            revenue=revenue,
            rows=rows,
            tag_example=tag_example,
            tag_host=tag_host,
            tag_links=tag_links,
            others=others,
            set_at=set_at,
            set_by=set_by,
        )

        return commerce_attribution


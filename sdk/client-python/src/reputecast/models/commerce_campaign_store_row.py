from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="CommerceCampaignStoreRow")



@_attrs_define
class CommerceCampaignStoreRow:
    """ 
        Attributes:
            by_email (int):
            by_token (int):
            currency (str):
            days (int):
            name (str):
            orders (int):
            platform (str):
            revenue (int):
            store_id (str):
     """

    by_email: int
    by_token: int
    currency: str
    days: int
    name: str
    orders: int
    platform: str
    revenue: int
    store_id: str





    def to_dict(self) -> dict[str, Any]:
        by_email = self.by_email

        by_token = self.by_token

        currency = self.currency

        days = self.days

        name = self.name

        orders = self.orders

        platform = self.platform

        revenue = self.revenue

        store_id = self.store_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "byEmail": by_email,
            "byToken": by_token,
            "currency": currency,
            "days": days,
            "name": name,
            "orders": orders,
            "platform": platform,
            "revenue": revenue,
            "storeId": store_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        by_email = d.pop("byEmail")

        by_token = d.pop("byToken")

        currency = d.pop("currency")

        days = d.pop("days")

        name = d.pop("name")

        orders = d.pop("orders")

        platform = d.pop("platform")

        revenue = d.pop("revenue")

        store_id = d.pop("storeId")

        commerce_campaign_store_row = cls(
            by_email=by_email,
            by_token=by_token,
            currency=currency,
            days=days,
            name=name,
            orders=orders,
            platform=platform,
            revenue=revenue,
            store_id=store_id,
        )

        return commerce_campaign_store_row


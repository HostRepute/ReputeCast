from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="CampaignProductSale")



@_attrs_define
class CampaignProductSale:
    """ 
        Attributes:
            name (str):
            orders (int):
            product_id (str):
            revenue_cents (int):
     """

    name: str
    orders: int
    product_id: str
    revenue_cents: int





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        orders = self.orders

        product_id = self.product_id

        revenue_cents = self.revenue_cents


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
            "orders": orders,
            "productId": product_id,
            "revenueCents": revenue_cents,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        orders = d.pop("orders")

        product_id = d.pop("productId")

        revenue_cents = d.pop("revenueCents")

        campaign_product_sale = cls(
            name=name,
            orders=orders,
            product_id=product_id,
            revenue_cents=revenue_cents,
        )

        return campaign_product_sale

